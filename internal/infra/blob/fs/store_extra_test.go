package fs

import (
	"colonycore/internal/blob/core"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStoreCRUD exercises Put/Get/Head/Delete/List and key sanitization errors.
func TestStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	// invalid keys
	for _, bad := range []string{"", "../escape", "/abs", "..", "a/../b"} {
		if _, _, e := st.pathFor(bad); e == nil {
			t.Fatalf("expected sanitize error for %q", bad)
		}
	}
	info, err := st.Put(context.Background(), "a/b.txt", strings.NewReader("hello"), core.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "a/b.txt" || info.Size != 5 {
		t.Fatalf("unexpected info: %+v", info)
	}
	if _, err := st.Put(context.Background(), "a/b.txt", strings.NewReader("dup"), core.PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	head, err := st.Head(context.Background(), "a/b.txt")
	if err != nil || head.ETag == "" {
		t.Fatalf("head: %+v %v", head, err)
	}
	gInfo, rc, err := st.Get(context.Background(), "a/b.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	b, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(b) != "hello" || gInfo.ETag != head.ETag {
		t.Fatalf("unexpected get result")
	}
	list, err := st.List(context.Background(), "a/")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v %v", list, err)
	}
	ok, err := st.Delete(context.Background(), "a/b.txt")
	if err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	if _, _, err := st.Get(context.Background(), "a/b.txt"); err == nil {
		t.Fatalf("expected get after delete error")
	}
	// ensure meta file removed
	dataPath := filepath.Join(dir, "a/b.txt")
	if _, err := os.Stat(dataPath + ".meta"); err == nil {
		t.Fatalf("expected meta gone")
	}
}
