package blob

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

func TestNewMockS3ForTestsBasic(t *testing.T) {
	s := NewMockS3ForTests()
	if s.Driver() != DriverS3 {
		t.Fatalf("expected DriverS3")
	}
	// Put then Get
	if _, err := s.Put(context.Background(), "a.txt", bytes.NewReader([]byte("hello")), PutOptions{ContentType: "text/plain"}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, rc, err := s.Get(context.Background(), "a.txt"); err != nil {
		t.Fatalf("get: %v", err)
	} else {
		_, _ = io.ReadAll(rc)
		_ = rc.Close()
	}
	if _, err := s.Head(context.Background(), "a.txt"); err != nil {
		t.Fatalf("head: %v", err)
	}
	if list, err := s.List(context.Background(), ""); err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
	if _, err := s.PresignURL(context.Background(), "a.txt", SignedURLOptions{}); err != nil {
		t.Fatalf("presign: %v", err)
	}
	if ok, err := s.Delete(context.Background(), "a.txt"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
}

func TestDecodeChunkedLiteAndParseHex(t *testing.T) {
	// Build single chunk payload of "abc"
	payload := []byte("3\r\nabc\r\n0\r\n")
	dec, ok := decodeChunkedLite(payload)
	if !ok || string(dec) != "abc" {
		t.Fatalf("unexpected decode: %v %q", ok, string(dec))
	}
	// Invalid hex
	if _, err := parseHex("zz"); err == nil {
		t.Fatalf("expected parseHex error")
	}
	// Invalid chunk (bad length)
	if _, ok := decodeChunkedLite([]byte("5\r\nabc\r\n0\r\n")); ok {
		t.Fatalf("expected decode failure")
	}
}

// Ensure RoundTrip 501 path covered with unsupported method
func TestMockRoundTripperLiteUnsupported(t *testing.T) {
	rt := &mockRoundTripperLite{state: make(map[string]mockObj)}
	req, _ := http.NewRequest(http.MethodPatch, "https://mock.s3.local/bucket/key", nil)
	resp, _ := rt.RoundTrip(req)
	if resp.StatusCode != 501 {
		t.Fatalf("expected 501, got %d", resp.StatusCode)
	}
}
