package memory

import (
	"bytes"
	"context"
	"io"
	"testing"

	coreblob "colonycore/internal/blob/core"
)

// TestStoreGetHeadNotFoundAndSuccess increases coverage for Get and Head branches.
func TestStoreGetHeadNotFoundAndSuccess(t *testing.T) {
	store := New()
	ctx := context.Background()
	// not found branches
	if _, _, err := store.Get(ctx, "missing"); err == nil {
		t.Fatalf("expected get missing error")
	}
	if _, err := store.Head(ctx, "missing"); err == nil {
		t.Fatalf("expected head missing error")
	}
	// put an object
	info, err := store.Put(ctx, "k1", bytes.NewBufferString("payload"), coreblob.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "k1" {
		t.Fatalf("unexpected key %s", info.Key)
	}
	// get it
	gotInfo, r, err := store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	b, _ := io.ReadAll(r)
	if string(b) != "payload" {
		t.Fatalf("unexpected payload %s", string(b))
	}
	if gotInfo.ContentType != "text/plain" || gotInfo.Size == 0 {
		t.Fatalf("unexpected info %+v", gotInfo)
	}
	// head it
	headInfo, err := store.Head(ctx, "k1")
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	if headInfo.Key != "k1" || headInfo.Size != gotInfo.Size {
		t.Fatalf("unexpected head info %+v", headInfo)
	}
}
