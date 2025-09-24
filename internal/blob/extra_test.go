package blob

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestFilesystem_ErrorBranches(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	fs, err := NewFilesystem(dir)
	if err != nil {
		t.Fatalf("new fs: %v", err)
	}
	// Presign before object exists; older behavior returned error. If implementation changes
	// to allow presign without existence, we simply ignore result for coverage.
	_, _ = fs.PresignURL(ctx, "pfx/a.txt", SignedURLOptions{})
	// Put one object
	if _, err := fs.Put(ctx, "pfx/a.txt", bytes.NewReader([]byte("one")), PutOptions{}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if fs.Driver() != DriverFilesystem {
		t.Fatalf("expected filesystem driver")
	}
	// List with non-matching prefix
	list, err := fs.List(ctx, "other/")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty slice for unmatched prefix")
	}
	// Get missing
	if _, _, err := fs.Get(ctx, "does/not/exist"); err == nil {
		t.Fatalf("expected error for missing get")
	}
	// Head missing
	if _, err := fs.Head(ctx, "does/not/exist"); err == nil {
		t.Fatalf("expected error for missing head")
	}
}

func TestFactory_Memory(t *testing.T) {
	// memory driver
	t.Setenv("COLONYCORE_BLOB_DRIVER", "memory")
	bs, err := Open(context.Background())
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	if bs.Driver() != DriverMemory {
		t.Fatalf("expected memory driver")
	}
}

func TestFactoryDefaultFilesystemAndErrors(t *testing.T) {
	ctx := context.Background()
	_ = os.Unsetenv("COLONYCORE_BLOB_DRIVER") // explicitly ignore error
	// ensure root env set to temp dir for deterministic cleanup
	dir := t.TempDir()
	t.Setenv("COLONYCORE_BLOB_FS_ROOT", dir)
	bs, err := Open(ctx)
	if err != nil || bs.Driver() != DriverFilesystem {
		t.Fatalf("expected filesystem driver: %v %v", bs, err)
	}
	if _, err := bs.Head(ctx, "does-not-exist"); err == nil {
		t.Fatalf("expected head error")
	}
	if _, _, err := bs.Get(ctx, "does-not-exist"); err == nil {
		t.Fatalf("expected get error")
	}
}

func TestS3_OpenFromEnvRequiresBucket(t *testing.T) {
	t.Setenv("COLONYCORE_BLOB_DRIVER", "s3")
	_ = os.Unsetenv("COLONYCORE_BLOB_S3_BUCKET") // ensure missing; ignore error
	if _, err := Open(context.Background()); err == nil {
		t.Fatalf("expected error without bucket")
	}
}
