package datasets

import (
	"context"
	"testing"
)

func TestMemoryObjectStorePutGetListDelete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryObjectStore()

	meta := map[string]any{"purpose": "test", "count": 1}
	a1, err := store.Put(ctx, "exp1/artifactA", []byte("hello"), "text/plain", meta)
	if err != nil {
		t.Fatalf("put a1: %v", err)
	}
	if a1.ID != "exp1/artifactA" || a1.SizeBytes != 5 {
		t.Fatalf("unexpected artifact metadata: %+v", a1)
	}

	// Ensure metadata map is defensively copied.
	meta["mutated"] = true
	gotMeta, payload, err := store.Get(ctx, "exp1/artifactA")
	if err != nil {
		t.Fatalf("get a1: %v", err)
	}
	if string(payload) != "hello" {
		t.Fatalf("expected payload 'hello', got %q", string(payload))
	}
	if _, ok := gotMeta.Metadata["mutated"]; ok {
		t.Fatalf("store metadata mutated via caller map")
	}

	// Second object
	if _, err := store.Put(ctx, "exp1/artifactB", []byte("world"), "text/plain", nil); err != nil {
		t.Fatalf("put a2: %v", err)
	}

	// List prefix
	list, err := store.List(ctx, "exp1/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(list))
	}

	// Delete one
	existed, err := store.Delete(ctx, "exp1/artifactA")
	if err != nil || !existed {
		t.Fatalf("delete existing: existed=%v err=%v", existed, err)
	}
	// Idempotent delete
	existed, err = store.Delete(ctx, "exp1/artifactA")
	if err != nil || existed {
		t.Fatalf("delete idempotent expected false,nil got %v,%v", existed, err)
	}

	// Get after delete should error
	if _, _, err := store.Get(ctx, "exp1/artifactA"); err == nil {
		t.Fatalf("expected error on deleted object")
	}

	// List should now have 1
	list, err = store.List(ctx, "exp1/")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 1 || list[0].ID != "exp1/artifactB" {
		t.Fatalf("expected only artifactB remaining, got %+v", list)
	}
}

func TestMemoryObjectStoreDuplicateKey(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryObjectStore()
	if _, err := store.Put(ctx, "dup", []byte("one"), "text/plain", nil); err != nil {
		t.Fatalf("put first: %v", err)
	}
	if _, err := store.Put(ctx, "dup", []byte("two"), "text/plain", nil); err == nil {
		t.Fatalf("expected error on duplicate key")
	}
}
