package memory

import (
	"bytes"
	"colonycore/internal/blob/core"
	"context"
	"testing"
)

func TestMemoryStoreCRUDAndPresign(t *testing.T) {
	st := New()
	if st.Driver() != core.DriverMemory {
		t.Fatalf("driver mismatch")
	}
	// put
	info, err := st.Put(context.Background(), "k1", bytes.NewReader([]byte("data")), core.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"x": "y"}})
	if err != nil || info.Key != "k1" {
		t.Fatalf("put failed: %+v %v", info, err)
	}
	// duplicate
	if _, err := st.Put(context.Background(), "k1", bytes.NewReader([]byte("d2")), core.PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	// head
	h, err := st.Head(context.Background(), "k1")
	if err != nil || h.Size != 4 {
		t.Fatalf("head failed: %+v %v", h, err)
	}
	// get
	gInfo, r, err := st.Get(context.Background(), "k1")
	if err != nil || gInfo.Size != 4 {
		t.Fatalf("get failed: %+v %v", gInfo, err)
	}
	_ = r.Close()
	// list prefix success
	items, err := st.List(context.Background(), "k")
	if err != nil || len(items) != 1 {
		t.Fatalf("list failed: %v %v", items, err)
	}
	// delete
	ok, err := st.Delete(context.Background(), "k1")
	if err != nil || !ok {
		t.Fatalf("delete failed: %v %v", ok, err)
	}
	// list empty
	items, _ = st.List(context.Background(), "k")
	if len(items) != 0 {
		t.Fatalf("expected empty after delete")
	}
	// unsupported presign
	if _, err := st.PresignURL(context.Background(), "k1", core.SignedURLOptions{}); err == nil {
		t.Fatalf("expected unsupported presign")
	}
}
