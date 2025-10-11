package fs

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	coreblob "colonycore/internal/blob/core"
)

// TestStoreDriverAndPresignAndDelete increases coverage for Driver and PresignURL branches.
func TestStoreDriverAndPresignAndDelete(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if store.Driver() != coreblob.DriverFilesystem {
		t.Fatalf("unexpected driver %v", store.Driver())
	}
	ctx := context.Background()
	// put object
	info, err := store.Put(ctx, "dir/file.txt", bytes.NewBufferString("data"), coreblob.PutOptions{ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "dir/file.txt" {
		t.Fatalf("unexpected key %s", info.Key)
	}
	// get and read
	_, rc, err := store.Get(ctx, info.Key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	b, _ := io.ReadAll(rc)
	if string(b) != "data" {
		t.Fatalf("unexpected payload %s", string(b))
	}
	// verify file physically exists
	if _, statErr := os.Stat(filepath.Join(root, "dir", "file.txt")); statErr != nil {
		t.Fatalf("expected file on disk: %v", statErr)
	}
	// presign GET supported
	if _, err := store.PresignURL(ctx, info.Key, coreblob.SignedURLOptions{Method: "GET"}); err != nil {
		t.Fatalf("presign GET: %v", err)
	}
	// presign PUT unsupported
	if _, err := store.PresignURL(ctx, info.Key, coreblob.SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected presign unsupported error")
	}
	// delete existing
	deleted, err := store.Delete(ctx, info.Key)
	if err != nil || !deleted {
		t.Fatalf("expected delete success, err=%v deleted=%v", err, deleted)
	}
	// delete missing returns false
	deleted, err = store.Delete(ctx, info.Key)
	if err != nil || deleted {
		t.Fatalf("expected delete false for missing, err=%v deleted=%v", err, deleted)
	}
}
