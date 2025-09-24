package blob

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	aws "github.com/aws/aws-sdk-go-v2/aws"
)

func TestFilesystem_PresignVariantsAndListOrder(t *testing.T) {
	ctx := context.Background()
	fs, err := NewFilesystem(t.TempDir())
	if err != nil {
		t.Fatalf("fs: %v", err)
	}
	if _, err := fs.Put(ctx, "a/1.txt", bytes.NewReader([]byte("a1")), PutOptions{}); err != nil {
		t.Fatalf("put1: %v", err)
	}
	if _, err := fs.Put(ctx, "b/2.txt", bytes.NewReader([]byte("b2")), PutOptions{}); err != nil {
		t.Fatalf("put2: %v", err)
	}
	// lower-case method should normalize
	if url, err := fs.PresignURL(ctx, "a/1.txt", SignedURLOptions{Method: "get"}); err != nil || url == "" {
		t.Fatalf("presign lower: %v %s", err, url)
	}
	// unsupported method
	if _, err := fs.PresignURL(ctx, "a/1.txt", SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected unsupported for PUT")
	}
	// list with empty prefix should include both keys (order sorted)
	list, err := fs.List(ctx, "")
	if err != nil || len(list) != 2 {
		t.Fatalf("list root: %v %v", err, list)
	}
	if list[0].Key > list[1].Key {
		t.Fatalf("expected sorted order: %+v", list)
	}
}

func TestS3_PresignCustomExpiryAndEmptyList(t *testing.T) {
	s3store := newMockS3(t)
	ctx := context.Background()
	if _, err := s3store.Put(ctx, "k.txt", bytes.NewReader([]byte("body")), PutOptions{}); err != nil {
		t.Fatalf("put: %v", err)
	}
	// custom expiry branch
	if url, err := s3store.PresignURL(ctx, "k.txt", SignedURLOptions{Expiry: 30 * time.Second}); err != nil || url == "" {
		t.Fatalf("presign custom: %v %s", err, url)
	}
	// empty list prefix
	if list, err := s3store.List(ctx, "no-such-prefix/"); err != nil || len(list) != 0 {
		t.Fatalf("expected empty list: %v %+v", err, list)
	}
	// pagination: add second object then list common prefix
	if _, err := s3store.Put(ctx, "k2.txt", bytes.NewReader([]byte("body2")), PutOptions{}); err != nil {
		t.Fatalf("put2: %v", err)
	}
	if list, err := s3store.List(ctx, "k"); err != nil || len(list) != 2 {
		t.Fatalf("expected two items via pagination: %v %+v", err, list)
	}
}

func TestMemory_MissingHeadGet(t *testing.T) {
	m := newMemoryStore()
	ctx := context.Background()
	if _, err := m.Head(ctx, "missing"); err == nil {
		t.Fatalf("expected head error")
	}
	if _, _, err := m.Get(ctx, "missing"); err == nil {
		t.Fatalf("expected get error")
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

func TestS3_FromHeadNilBranchesAndErrors(t *testing.T) {
	s3store := newMockS3(t)
	// fromHead with nil optional fields (contentType nil, lastModified nil) and etag trimming
	info := s3store.fromHead("k", 10, nil, aws.String("\"etagval\""), map[string]string{"x": "y"}, nil)
	if info.ETag != "etagval" || info.ContentType != "" || info.Key != "k" || info.Size != 10 {
		t.Fatalf("unexpected info: %+v", info)
	}
	// NewS3 error path (missing bucket)
	if _, err := NewS3(context.Background(), S3Config{}); err == nil {
		t.Fatalf("expected error for missing bucket")
	}
}

func TestFilesystem_NewFilesystemFileError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := NewFilesystem(filePath); err == nil {
		t.Fatalf("expected error when root is file")
	}
}
