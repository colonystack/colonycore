package datasets

import (
	"context"
	"testing"
)

func TestMemoryObjectStoreCRUD(t *testing.T) {
	store := NewMemoryObjectStore()
	art, err := store.Put(context.Background(), "obj1", []byte("payload"), "text/plain", map[string]any{"a": 1})
	if err != nil || art.ID != "obj1" || art.SizeBytes != 7 {
		t.Fatalf("put failed: %+v %v", art, err)
	}
	// duplicate
	if _, err := store.Put(context.Background(), "obj1", []byte("again"), "text/plain", nil); err == nil {
		t.Fatalf("expected duplicate error")
	}
	// get
	gart, body, err := store.Get(context.Background(), "obj1")
	if err != nil || string(body) != "payload" || gart.ID != "obj1" {
		t.Fatalf("get mismatch: %+v %s %v", gart, string(body), err)
	}
	// list
	lst, err := store.List(context.Background(), "obj")
	if err != nil || len(lst) != 1 {
		t.Fatalf("list mismatch: %v %v", lst, err)
	}
	// delete
	ok, err := store.Delete(context.Background(), "obj1")
	if err != nil || !ok {
		t.Fatalf("delete failed: %v %v", ok, err)
	}
	// ensure gone
	if _, _, err := store.Get(context.Background(), "obj1"); err == nil {
		t.Fatalf("expected get missing error")
	}
}
