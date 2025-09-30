package memory

import (
	"bytes"
	"colonycore/internal/blob/core"
	"context"
	"fmt"
	"testing"
)

func TestStore_MissingHeadGet(t *testing.T) {
	store := New()
	ctx := context.Background()
	if _, err := store.Head(ctx, "missing"); err == nil {
		t.Fatalf("expected head error")
	}
	if _, _, err := store.Get(ctx, "missing"); err == nil {
		t.Fatalf("expected get error")
	}
}

func TestStore_AllBranches(t *testing.T) {
	store := New()
	ctx := context.Background()
	if _, _, err := store.Get(ctx, "missing"); err == nil {
		t.Fatalf("expected missing get error")
	}
	if _, err := store.Head(ctx, "missing"); err == nil {
		t.Fatalf("expected missing head error")
	}
	if ok, err := store.Delete(ctx, "missing"); err != nil || ok {
		t.Fatalf("expected delete false")
	}
	if _, err := store.Put(ctx, "k", bytes.NewReader([]byte("v")), core.PutOptions{Metadata: map[string]string{"a": "1"}}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if _, err := store.Put(ctx, "k", bytes.NewReader([]byte("v2")), core.PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	if list, err := store.List(ctx, ""); err != nil || len(list) != 1 {
		t.Fatalf("list all: %v %d", err, len(list))
	}
	if list, err := store.List(ctx, "k"); err != nil || len(list) != 1 {
		t.Fatalf("list prefix: %v %d", err, len(list))
	}
	if _, err := store.PresignURL(ctx, "k", core.SignedURLOptions{}); err == nil {
		t.Fatalf("expected unsupported presign")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, fmt.Errorf("fail") }

func TestStore_PutReadErrorAndDriver(t *testing.T) {
	store := New()
	if store.Driver() != core.DriverMemory {
		t.Fatalf("expected memory driver")
	}
	if _, err := store.Put(context.Background(), "bad", failingReader{}, core.PutOptions{}); err == nil {
		t.Fatalf("expected read error")
	}
}
