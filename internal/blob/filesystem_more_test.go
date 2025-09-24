package blob

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// errorReader triggers an error mid copy for Put error branch.
type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) { return 0, errors.New("boom") }

func TestFilesystem_PutDuplicateAndErrorBranches(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(dir)
	if err != nil {
		t.Fatalf("new fs: %v", err)
	}
	// successful put
	if _, err := fs.Put(context.Background(), "k1.txt", bytes.NewReader([]byte("hi")), PutOptions{ContentType: "text/plain", Metadata: map[string]string{"a": "1"}}); err != nil {
		t.Fatalf("put1: %v", err)
	}
	// duplicate
	if _, err := fs.Put(context.Background(), "k1.txt", bytes.NewReader([]byte("again")), PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	// error reader path
	if _, err := fs.Put(context.Background(), "bad.bin", errorReader{}, PutOptions{}); err == nil {
		t.Fatalf("expected copy error")
	}
}

func TestFilesystem_HeadGetDeleteAndList(t *testing.T) { //nolint:cyclop
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	ctx := context.Background()
	// Put multiple
	for i := 0; i < 3; i++ {
		k := filepath.Join("folder", "f"+strconv.Itoa(i)+".txt")
		if _, err := fs.Put(ctx, k, bytes.NewReader([]byte("data")), PutOptions{}); err != nil {
			t.Fatalf("put %s: %v", k, err)
		}
	}
	// Head success
	if _, err := fs.Head(ctx, "folder/f0.txt"); err != nil {
		t.Fatalf("head: %v", err)
	}
	// Get success
	if _, rc, err := fs.Get(ctx, "folder/f1.txt"); err != nil {
		t.Fatalf("get: %v", err)
	} else {
		_ = rc.Close()
	}
	// Delete existing
	if ok, err := fs.Delete(ctx, "folder/f2.txt"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	// Delete missing
	if ok, err := fs.Delete(ctx, "folder/missing.txt"); err != nil || ok {
		t.Fatalf("expected delete false")
	}
	// List prefix
	list, err := fs.List(ctx, "folder/")
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	// Get meta-missing error path: remove meta file manually
	_, metaPath, _ := fs.pathFor("folder/f0.txt")
	if err := os.Remove(metaPath); err != nil {
		t.Fatalf("rm meta: %v", err)
	}
	if _, _, err := fs.Get(ctx, "folder/f0.txt"); err == nil {
		t.Fatalf("expected get meta error")
	}
	if _, err := fs.Head(ctx, "folder/f0.txt"); err == nil {
		t.Fatalf("expected head meta error")
	}
}

func TestFilesystem_PresignUnsupported(t *testing.T) {
	dir := t.TempDir()
	fs, _ := NewFilesystem(dir)
	if _, err := fs.PresignURL(context.Background(), "some", SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected unsupported error")
	}
}

func TestMemoryStore_AllBranches(t *testing.T) {
	m := newMemoryStore()
	ctx := context.Background()
	if _, _, err := m.Get(ctx, "missing"); err == nil {
		t.Fatalf("expected missing get error")
	}
	if _, err := m.Head(ctx, "missing"); err == nil {
		t.Fatalf("expected missing head error")
	}
	if ok, err := m.Delete(ctx, "missing"); err != nil || ok {
		t.Fatalf("expected delete false")
	}
	if _, err := m.Put(ctx, "k", bytes.NewReader([]byte("v")), PutOptions{Metadata: map[string]string{"a": "1"}}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, err := m.Put(ctx, "k", bytes.NewReader([]byte("v2")), PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	if list, err := m.List(ctx, ""); err != nil || len(list) != 1 {
		t.Fatalf("list all: %v %d", err, len(list))
	}
	if list, err := m.List(ctx, "k"); err != nil || len(list) != 1 {
		t.Fatalf("list prefix: %v %d", err, len(list))
	}
	if _, err := m.PresignURL(ctx, "k", SignedURLOptions{}); err == nil {
		t.Fatalf("expected unsupported presign")
	}
}

func TestS3_FromHeadHelper(t *testing.T) {
	s := &S3{}
	lm := time.Now().Add(-time.Hour)
	info := s.fromHead("k", 5, strPtr("text/plain"), strPtr("\"etag123\""), map[string]string{"m": "1"}, &lm)
	if info.ContentType != "text/plain" || info.ETag != "etag123" || info.Size != 5 || info.LastModified != lm {
		t.Fatalf("unexpected info: %#v", info)
	}
	// nil lastModified path
	info2 := s.fromHead("k2", 1, nil, nil, nil, nil)
	if info2.ETag != "" || info2.ContentType != "" || info2.LastModified.IsZero() { // LastModified should be now-ish (non-zero)
		t.Fatalf("unexpected info2: %#v", info2)
	}
}

func strPtr(s string) *string { return &s }
