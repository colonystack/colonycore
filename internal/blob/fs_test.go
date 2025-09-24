package blob

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func newTempFS(t *testing.T) *Filesystem {
	dir := t.TempDir()
	fs, err := NewFilesystem(dir)
	if err != nil {
		t.Fatalf("NewFilesystem: %v", err)
	}
	return fs
}

func TestFilesystem_PutGetHeadListDelete(t *testing.T) {
	ctx := context.Background()
	fs := newTempFS(t)
	info, err := fs.Put(ctx, "alpha/test.txt", bytes.NewReader([]byte("hello")), PutOptions{ContentType: "text/plain", Metadata: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "alpha/test.txt" || info.Size != 5 {
		t.Fatalf("unexpected info %+v", info)
	}
	// duplicate should fail
	if _, err := fs.Put(ctx, "alpha/test.txt", bytes.NewReader([]byte("x")), PutOptions{}); err == nil {
		t.Fatalf("expected duplicate failure")
	}
	// Head
	h, err := fs.Head(ctx, "alpha/test.txt")
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if h.ETag == "" {
		t.Fatalf("etag expected")
	}
	// Get
	g, rc, err := fs.Get(ctx, "alpha/test.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	b, _ := io.ReadAll(rc)
	if err := rc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if string(b) != "hello" || g.ETag != h.ETag {
		t.Fatalf("unexpected get artifacts")
	}
	// List prefix
	list, err := fs.List(ctx, "alpha/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "alpha/test.txt" {
		t.Fatalf("unexpected list %+v", list)
	}
	// Presign
	url, err := fs.PresignURL(ctx, "alpha/test.txt", SignedURLOptions{})
	if err != nil || url == "" {
		t.Fatalf("presign url: %v %s", err, url)
	}
	// Delete
	ok, err := fs.Delete(ctx, "alpha/test.txt")
	if err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	ok, err = fs.Delete(ctx, "alpha/test.txt")
	if err != nil || ok {
		t.Fatalf("second delete should be false")
	}
}

func TestFilesystem_PathTraversal(t *testing.T) {
	ctx := context.Background()
	fs := newTempFS(t)
	_, err := fs.Put(ctx, "../escape.txt", bytes.NewReader([]byte("x")), PutOptions{})
	if err == nil {
		t.Fatalf("expected traversal error")
	}
	_, err = fs.Put(ctx, "/abs.txt", bytes.NewReader([]byte("x")), PutOptions{})
	if err == nil {
		t.Fatalf("expected absolute error")
	}
}

func TestFilesystem_MetadataPersistence(t *testing.T) {
	ctx := context.Background()
	fs := newTempFS(t)
	_, err := fs.Put(ctx, "meta/data.bin", bytes.NewReader([]byte("abc")), PutOptions{ContentType: "application/octet-stream", Metadata: map[string]string{"a": "1"}})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	// inspect raw meta file
	dataPath, metaPath, _ := fs.pathFor("meta/data.bin")
	if _, err := os.Stat(dataPath); err != nil {
		t.Fatalf("expected data path")
	}
	b, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if !bytes.Contains(b, []byte("application/octet-stream")) {
		t.Fatalf("meta missing content type")
	}
	if filepath.Ext(metaPath) != ".meta" {
		t.Fatalf("meta path extension mismatch")
	}
}
