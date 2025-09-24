package blob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// mockRoundTripper provides a tiny fake S3 subset sufficient to exercise our S3 adapter
// without network access. It stores objects in-memory keyed by object key.
type mockRoundTripper struct{ state map[string]stored }

type stored struct {
	body        []byte
	contentType string
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { //nolint:cyclop
	// Expect path-style: /bucket/key
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	key := ""
	if len(parts) == 2 {
		key = parts[1]
	}
	// Handle ListObjectsV2 (list-type=2)
	if req.Method == http.MethodGet && strings.Contains(req.URL.RawQuery, "list-type=2") {
		prefix := req.URL.Query().Get("prefix")
		cont := req.URL.Query().Get("continuation-token")
		// Collect & sort keys for deterministic pagination.
		var keys []string
		for k := range m.state {
			if prefix == "" || strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("<?xml version=\"1.0\"?><ListBucketResult>")
		if cont == "" && len(keys) > 1 {
			// First page: return first key only, truncated
			k := keys[0]
			st := m.state[k]
			b.WriteString("<IsTruncated>true</IsTruncated><NextContinuationToken>tok123</NextContinuationToken>")
			b.WriteString("<Contents><Key>")
			b.WriteString(k)
			b.WriteString("</Key><Size>")
			b.WriteString(fmt.Sprintf("%d", len(st.body)))
			b.WriteString("</Size><LastModified>2024-01-01T00:00:00Z</LastModified></Contents>")
		} else {
			b.WriteString("<IsTruncated>false</IsTruncated>")
			// Second (or single) page: if continuation token provided skip first key
			start := 0
			if cont != "" && len(keys) > 1 {
				start = 1
			}
			for _, k := range keys[start:] {
				st := m.state[k]
				b.WriteString("<Contents><Key>")
				b.WriteString(k)
				b.WriteString("</Key><Size>")
				b.WriteString(fmt.Sprintf("%d", len(st.body)))
				b.WriteString("</Size><LastModified>2024-01-01T00:00:00Z</LastModified></Contents>")
			}
		}
		b.WriteString("</ListBucketResult>")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(b.String())), Header: http.Header{"Content-Type": {"application/xml"}}}, nil
	}
	switch req.Method {
	case http.MethodHead:
		if st, ok := m.state[key]; ok { //nolint:revive // test helper
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{
				"Content-Length": {fmt.Sprintf("%d", len(st.body))},
				"Content-Type":   {st.contentType},
				"ETag":           {"\"etag123\""},
				"Last-Modified":  {time.Now().UTC().Format(http.TimeFormat)},
			}}, nil
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	case http.MethodPut:
		body, _ := io.ReadAll(req.Body)
		if _, exists := m.state[key]; !exists { // emulate create-only semantics
			ct := req.Header.Get("Content-Type")
			if dec, ok := decodeChunked(body); ok { // handle aws-chunked encoding
				body = dec
			}
			m.state[key] = stored{body: body, contentType: ct}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{"ETag": {"\"etag\""}}}, nil
	case http.MethodGet:
		if st, ok := m.state[key]; ok {
			body := st.body
			if dec, ok := decodeChunked(body); ok {
				body = dec
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{
				"Content-Length": {fmt.Sprintf("%d", len(body))},
				"Content-Type":   {st.contentType},
				"Last-Modified":  {time.Now().UTC().Format(http.TimeFormat)},
				"ETag":           {"\"etag\""},
			}}, nil
		}
		return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	case http.MethodDelete:
		delete(m.state, key)
		return &http.Response{StatusCode: 204, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	}
	return &http.Response{StatusCode: 501, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
}

func newMockS3(t *testing.T) *S3 {
	t.Helper()
	rt := &mockRoundTripper{state: make(map[string]stored)}
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIA", "SECRET", "")),
	)
	if err != nil {
		t.Fatalf("cfg: %v", err)
	}
	httpClient := &http.Client{Transport: rt}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://mock.s3.local")
		o.HTTPClient = httpClient
		o.UsePathStyle = true
	})
	ps := s3.NewPresignClient(client)
	return &S3{client: client, bucket: "test-bucket", presign: ps}
}

func TestS3_MockedBasicFlow(t *testing.T) {
	s3store := newMockS3(t)
	ctx := context.Background()
	info, err := s3store.Put(ctx, "folder/file.txt", bytes.NewReader([]byte("hello")), PutOptions{ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "folder/file.txt" || info.ContentType != "text/plain" || info.Size < 5 {
		t.Fatalf("unexpected info %#v", info)
	}
	// Duplicate put should fail (coverage of exists branch)
	if _, err := s3store.Put(ctx, "folder/file.txt", bytes.NewReader([]byte("ignored")), PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	if _, err := s3store.Head(ctx, "folder/file.txt"); err != nil {
		t.Fatalf("head: %v", err)
	}
	_, rc, err := s3store.Get(ctx, "folder/file.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(data) != "hello" {
		t.Fatalf("get mismatch: %q", string(data))
	}
	list, err := s3store.List(ctx, "folder/")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	if url, err := s3store.PresignURL(ctx, "folder/file.txt", SignedURLOptions{}); err != nil || url == "" {
		t.Fatalf("presign: %v %s", err, url)
	}
	if ok, err := s3store.Delete(ctx, "folder/file.txt"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
}

func TestS3_NewS3(t *testing.T) {
	// Provide dummy creds so default chain resolves immediately.
	_ = os.Setenv("AWS_ACCESS_KEY_ID", "AKIA")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")
	defer func() {
		_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
		_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()
	s, err := NewS3(context.Background(), S3Config{Bucket: "bkt", Region: "us-east-1", Endpoint: "https://mock.s3.local", PathStyle: true})
	if err != nil {
		t.Fatalf("NewS3: %v", err)
	}
	if s.Driver() != DriverS3 {
		t.Fatalf("expected DriverS3")
	}
}

func TestS3_OpenFromEnv_Minimal(t *testing.T) {
	oldB := os.Getenv("COLONYCORE_BLOB_S3_BUCKET")
	oldR := os.Getenv("COLONYCORE_BLOB_S3_REGION")
	_ = os.Setenv("COLONYCORE_BLOB_S3_BUCKET", "env-bucket")
	_ = os.Setenv("COLONYCORE_BLOB_S3_REGION", "us-east-1")
	defer func() {
		_ = os.Setenv("COLONYCORE_BLOB_S3_BUCKET", oldB)
		_ = os.Setenv("COLONYCORE_BLOB_S3_REGION", oldR)
	}()
	// Ensure call returns error only if bucket missing (we provided it) — we ignore returned *S3 for coverage.
	if _, err := OpenFromEnv(context.Background()); err != nil {
		t.Fatalf("OpenFromEnv: %v", err)
	}
}

func TestS3_ErrorPaths(t *testing.T) {
	s3store := newMockS3(t)
	ctx := context.Background()
	// Head missing
	if _, err := s3store.Head(ctx, "nope"); err == nil {
		t.Fatalf("expected head error for missing key")
	}
	if _, _, err := s3store.Get(ctx, "nope"); err == nil {
		t.Fatalf("expected get error for missing key")
	}
	// Unsupported method presign
	if _, err := s3store.PresignURL(ctx, "k", SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected presign unsupported error")
	}
}

func TestFactoryAndMemoryPaths(t *testing.T) {
	ctx := context.Background()
	// Default (fs) path uses temp dir override
	_ = os.Setenv("COLONYCORE_BLOB_DRIVER", "memory")
	m, err := Open(ctx)
	if err != nil || m.Driver() != DriverMemory {
		t.Fatalf("expected memory driver: %v %v", m, err)
	}
	if _, err := m.PresignURL(ctx, "k", SignedURLOptions{}); err == nil { // memory unsupported
		t.Fatalf("expected presign unsupported for memory")
	}
	if list, err := m.List(ctx, ""); err != nil || len(list) != 0 { // empty list path
		t.Fatalf("expected empty list")
	}
	if ok, err := m.Delete(ctx, "missing"); err != nil || ok { // delete missing path
		t.Fatalf("expected delete false on missing")
	}
	// Unknown driver path
	_ = os.Setenv("COLONYCORE_BLOB_DRIVER", "unknown-driver")
	if _, err := Open(ctx); err == nil {
		t.Fatalf("expected error for unknown driver")
	}
	_ = os.Unsetenv("COLONYCORE_BLOB_DRIVER")
}

// decodeChunked attempts to parse a simple hex-size CRLF body CRLF 0 CRLF trailer pattern.
// Returns decoded body and true on success; otherwise (nil,false).
func decodeChunked(b []byte) ([]byte, bool) { // minimal implementation for test use only
	s := string(b)
	// Expect format: <hex>\r\n<payload>\r\n0\r\n...
	parts := strings.Split(s, "\r\n")
	if len(parts) < 3 {
		return nil, false
	}
	sizeHex := parts[0]
	n, err := strconv.ParseInt(sizeHex, 16, 64)
	if err != nil || n <= 0 {
		return nil, false
	}
	if int64(len(parts[1])) != n { // payload size mismatch
		return nil, false
	}
	if parts[2] != "0" { // must terminate
		return nil, false
	}
	return []byte(parts[1]), true
}
