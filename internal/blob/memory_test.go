package blob

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
)

func TestMemoryStore_Basic(t *testing.T) {
	bs := newMemoryStore()
	ctx := context.Background()
	info, err := bs.Put(ctx, "k1", bytes.NewReader([]byte("data")), PutOptions{ContentType: "text/plain", Metadata: map[string]string{"m": "1"}})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "k1" || info.Size != 4 {
		t.Fatalf("unexpected info %#v", info)
	}
	// duplicate
	if _, err := bs.Put(ctx, "k1", bytes.NewReader([]byte("x")), PutOptions{}); err == nil {
		t.Fatalf("expected duplicate error")
	}
	// head
	h, err := bs.Head(ctx, "k1")
	if err != nil || h.ETag != "" {
		t.Fatalf("head unexpected: %#v %v", h, err)
	}
	// get
	g, rc, err := bs.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	b, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(b) != "data" || g.Size != 4 {
		t.Fatalf("bad payload")
	}
	// list
	list, err := bs.List(ctx, "k")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %+v", err, list)
	}
	// list unmatched prefix
	if list2, err := bs.List(ctx, "zzz"); err != nil || len(list2) != 0 {
		t.Fatalf("expected empty list for unmatched prefix")
	}
	// delete
	ok, err := bs.Delete(ctx, "k1")
	if err != nil || !ok {
		t.Fatalf("delete expected true")
	}
	ok, _ = bs.Delete(ctx, "k1")
	if ok {
		t.Fatalf("second delete should be false")
	}
}

func TestFactory_InvalidDriver(t *testing.T) {
	// set env temporarily
	old := getenv("COLONYCORE_BLOB_DRIVER")
	setenv("COLONYCORE_BLOB_DRIVER", "invalid")
	defer setenv("COLONYCORE_BLOB_DRIVER", old)
	if _, err := Open(context.Background()); err == nil {
		t.Fatalf("expected error for invalid driver")
	}
}

// small indirections to avoid importing os in each test
func getenv(k string) string { return os.Getenv(k) }
func setenv(k, v string) {
	if v == "" {
		_ = os.Unsetenv(k)
	} else {
		_ = os.Setenv(k, v)
	}
}
