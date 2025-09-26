package s3

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
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"colonycore/internal/blob/core"
)

// mockRoundTripper provides a tiny fake S3 subset sufficient to exercise the adapter without network access.
type mockRoundTripper struct{ state map[string]stored }

type stored struct {
	body        []byte
	contentType string
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { //nolint:cyclop
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	key := ""
	if len(parts) == 2 {
		key = parts[1]
	}
	if req.Method == http.MethodGet && strings.Contains(req.URL.RawQuery, "list-type=2") {
		prefix := req.URL.Query().Get("prefix")
		cont := req.URL.Query().Get("continuation-token")
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
		if st, ok := m.state[key]; ok {
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
		if _, exists := m.state[key]; !exists {
			ct := req.Header.Get("Content-Type")
			if dec, ok := decodeChunked(body); ok {
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

func newMockStore(t *testing.T) *Store {
	t.Helper()
	rt := &mockRoundTripper{state: make(map[string]stored)}
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIA", "SECRET", "")),
	)
	if err != nil {
		t.Fatalf("cfg: %v", err)
	}
	client := awsS3.NewFromConfig(cfg, func(o *awsS3.Options) {
		o.BaseEndpoint = aws.String("https://mock.s3.local")
		o.HTTPClient = &http.Client{Transport: rt}
		o.UsePathStyle = true
	})
	ps := awsS3.NewPresignClient(client)
	return &Store{client: client, bucket: "test-bucket", presign: ps}
}

func TestStore_MockedBasicFlow(t *testing.T) {
	store := newMockStore(t)
	ctx := context.Background()
	info, err := store.Put(ctx, "folder/file.txt", bytes.NewReader([]byte("hello")), core.PutOptions{ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "folder/file.txt" || info.ContentType != "text/plain" || info.Size < 5 {
		t.Fatalf("unexpected info %#v", info)
	}
	if _, err := store.Put(ctx, "folder/file.txt", bytes.NewReader([]byte("ignored")), core.PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	if _, err := store.Head(ctx, "folder/file.txt"); err != nil {
		t.Fatalf("head: %v", err)
	}
	_, rc, err := store.Get(ctx, "folder/file.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(data) != "hello" {
		t.Fatalf("get mismatch: %q", string(data))
	}
	list, err := store.List(ctx, "folder/")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	if url, err := store.PresignURL(ctx, "folder/file.txt", core.SignedURLOptions{}); err != nil || url == "" {
		t.Fatalf("presign: %v %s", err, url)
	}
	if ok, err := store.Delete(ctx, "folder/file.txt"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
}

func TestStore_New(t *testing.T) {
	_ = os.Setenv("AWS_ACCESS_KEY_ID", "AKIA")
	_ = os.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")
	defer func() {
		_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
		_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()
	s, err := New(context.Background(), Config{Bucket: "bkt", Region: "us-east-1", Endpoint: "https://mock.s3.local", PathStyle: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Driver() != core.DriverS3 {
		t.Fatalf("expected DriverS3")
	}
}

func TestStore_OpenFromEnv_Minimal(t *testing.T) {
	oldB := os.Getenv("COLONYCORE_BLOB_S3_BUCKET")
	oldR := os.Getenv("COLONYCORE_BLOB_S3_REGION")
	_ = os.Setenv("COLONYCORE_BLOB_S3_BUCKET", "env-bucket")
	_ = os.Setenv("COLONYCORE_BLOB_S3_REGION", "us-east-1")
	defer func() {
		_ = os.Setenv("COLONYCORE_BLOB_S3_BUCKET", oldB)
		_ = os.Setenv("COLONYCORE_BLOB_S3_REGION", oldR)
	}()
	if _, err := OpenFromEnv(context.Background()); err != nil {
		t.Fatalf("OpenFromEnv: %v", err)
	}
}

func TestStore_ErrorPaths(t *testing.T) {
	store := newMockStore(t)
	ctx := context.Background()
	if _, err := store.Head(ctx, "nope"); err == nil {
		t.Fatalf("expected head error for missing key")
	}
	if _, _, err := store.Get(ctx, "nope"); err == nil {
		t.Fatalf("expected get error for missing key")
	}
	if _, err := store.PresignURL(ctx, "k", core.SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected presign unsupported error")
	}
}

func TestStore_PresignCustomExpiryAndEmptyList(t *testing.T) {
	store := newMockStore(t)
	ctx := context.Background()
	if _, err := store.Put(ctx, "k.txt", bytes.NewReader([]byte("body")), core.PutOptions{}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if url, err := store.PresignURL(ctx, "k.txt", core.SignedURLOptions{Expiry: 30 * time.Second}); err != nil || url == "" {
		t.Fatalf("presign custom: %v %s", err, url)
	}
	if list, err := store.List(ctx, "no-such-prefix/"); err != nil || len(list) != 0 {
		t.Fatalf("expected empty list: %v %+v", err, list)
	}
	if _, err := store.Put(ctx, "k2.txt", bytes.NewReader([]byte("body2")), core.PutOptions{}); err != nil {
		t.Fatalf("put2: %v", err)
	}
	if list, err := store.List(ctx, "k"); err != nil || len(list) != 2 {
		t.Fatalf("expected two items via pagination: %v %+v", err, list)
	}
}

func TestStore_FromHeadNilBranchesAndErrors(t *testing.T) {
	store := newMockStore(t)
	info := store.fromHead("k", 10, nil, aws.String("\"etagval\""), map[string]string{"x": "y"}, nil)
	if info.ETag != "etagval" || info.ContentType != "" || info.Key != "k" || info.Size != 10 {
		t.Fatalf("unexpected info: %+v", info)
	}
	if _, err := New(context.Background(), Config{}); err == nil {
		t.Fatalf("expected error for missing bucket")
	}
}

func TestNewMockForTestsBasic(t *testing.T) {
	store := NewMockForTests()
	if store.Driver() != core.DriverS3 {
		t.Fatalf("expected DriverS3")
	}
	if _, err := store.Put(context.Background(), "a.txt", bytes.NewReader([]byte("hello")), core.PutOptions{ContentType: "text/plain"}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, rc, err := store.Get(context.Background(), "a.txt"); err != nil {
		t.Fatalf("get: %v", err)
	} else {
		_, _ = io.ReadAll(rc)
		_ = rc.Close()
	}
	if _, err := store.Head(context.Background(), "a.txt"); err != nil {
		t.Fatalf("head: %v", err)
	}
	if list, err := store.List(context.Background(), ""); err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if _, err := store.PresignURL(context.Background(), "a.txt", core.SignedURLOptions{}); err != nil {
		t.Fatalf("presign: %v", err)
	}
	if ok, err := store.Delete(context.Background(), "a.txt"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
}

func TestDecodeChunkedHelper(t *testing.T) {
	if _, ok := decodeChunked([]byte("not-chunked")); ok {
		t.Fatalf("expected fail 1")
	}
	if _, ok := decodeChunked([]byte("5\r\nabc\r\n0\r\n")); ok {
		t.Fatalf("size mismatch should fail")
	}
	if b, ok := decodeChunked([]byte("5\r\nhello\r\n0\r\n")); !ok || string(b) != "hello" {
		t.Fatalf("expected decode hello")
	}
}

func TestMockRoundTripperLiteUnsupported(t *testing.T) {
	rt := &mockRoundTripperLite{state: make(map[string]mockObj)}
	req, _ := http.NewRequest(http.MethodPatch, "https://mock.s3.local/bucket/key", nil)
	resp, _ := rt.RoundTrip(req)
	if resp.StatusCode != 501 {
		t.Fatalf("expected 501, got %d", resp.StatusCode)
	}
}

func decodeChunked(b []byte) ([]byte, bool) {
	s := string(b)
	parts := strings.Split(s, "\r\n")
	if len(parts) < 3 {
		return nil, false
	}
	sizeHex := parts[0]
	n, err := strconv.ParseInt(sizeHex, 16, 64)
	if err != nil || n <= 0 {
		return nil, false
	}
	if int64(len(parts[1])) != n {
		return nil, false
	}
	if parts[2] != "0" {
		return nil, false
	}
	return []byte(parts[1]), true
}
