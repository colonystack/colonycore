package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewMockForTests returns an *Store backed by an in-memory fake HTTP transport.
// Only a subset of S3 operations required by the blob.Store interface are implemented.
func NewMockForTests() *Store {
	rt := &mockRoundTripperLite{state: make(map[string]mockObj)}
	cfg, _ := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKIA", "SECRET", "")),
	)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = &http.Client{Transport: rt}
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String("https://mock.s3.local")
	})
	ps := s3.NewPresignClient(client)
	return &Store{client: client, bucket: "mock-bucket", presign: ps}
}

// mockRoundTripperLite is a trimmed version of the more exhaustive mock in tests; it handles Head/Get/Put/Delete/ListObjectsV2.
type mockRoundTripperLite struct{ state map[string]mockObj }

type mockObj struct {
	body        []byte
	contentType string
}

func (m *mockRoundTripperLite) RoundTrip(req *http.Request) (*http.Response, error) { //nolint:cyclop
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	key := ""
	if len(parts) == 2 {
		key = parts[1]
	}
	if req.Method == http.MethodGet && strings.Contains(req.URL.RawQuery, "list-type=2") {
		prefix := req.URL.Query().Get("prefix")
		var keys []string
		for k := range m.state {
			if prefix == "" || strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("<?xml version=\"1.0\"?><ListBucketResult><IsTruncated>false</IsTruncated>")
		for _, k := range keys {
			st := m.state[k]
			b.WriteString("<Contents><Key>")
			b.WriteString(k)
			b.WriteString("</Key><Size>")
			b.WriteString(fmt.Sprintf("%d", len(st.body)))
			b.WriteString("</Size><LastModified>2024-01-01T00:00:00Z</LastModified></Contents>")
		}
		b.WriteString("</ListBucketResult>")
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(b.String())), Header: http.Header{"Content-Type": {"application/xml"}}}, nil
	}
	switch req.Method {
	case http.MethodHead:
		if st, ok := m.state[key]; ok {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{
				"Content-Length": {fmt.Sprintf("%d", len(st.body))},
				"Content-Type":   {st.contentType},
				"ETag":           {"\"etag123\""},
				"Last-Modified":  {time.Now().UTC().Format(http.TimeFormat)},
			}}, nil
		}
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	case http.MethodPut:
		body, _ := io.ReadAll(req.Body)
		if dec, ok := decodeChunkedLite(body); ok { // handle aws-chunked encoding
			body = dec
		}
		if _, exists := m.state[key]; !exists {
			m.state[key] = mockObj{body: body, contentType: req.Header.Get("Content-Type")}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{"ETag": {"\"etag\""}}}, nil
	case http.MethodGet:
		if st, ok := m.state[key]; ok {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(st.body)), Header: http.Header{
				"Content-Length": {fmt.Sprintf("%d", len(st.body))},
				"Content-Type":   {st.contentType},
				"Last-Modified":  {time.Now().UTC().Format(http.TimeFormat)},
				"ETag":           {"\"etag\""},
			}}, nil
		}
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	case http.MethodDelete:
		delete(m.state, key)
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	}
	return &http.Response{StatusCode: http.StatusNotImplemented, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
}

// decodeChunkedLite decodes a minimal single-chunk aws-chunked style payload: <hex>\r\n<body>\r\n0\r\n...
func decodeChunkedLite(b []byte) ([]byte, bool) {
	s := string(b)
	parts := strings.Split(s, "\r\n")
	if len(parts) < 3 {
		return nil, false
	}
	sizeHex := parts[0]
	sz, perr := parseHex(sizeHex)
	if perr != nil || int64(len(parts[1])) != sz || parts[2] != "0" {
		return nil, false
	}
	return []byte(parts[1]), true
}

func parseHex(h string) (int64, error) {
	var v int64
	for _, c := range h {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v += int64(c - '0')
		case c >= 'a' && c <= 'f':
			v += int64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v += int64(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex")
		}
	}
	return v, nil
}
