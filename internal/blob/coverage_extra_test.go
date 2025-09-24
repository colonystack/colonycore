package blob

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestSanitizeKeyErrors exercises invalid key paths.
func TestSanitizeKeyErrors(t *testing.T) {
	cases := []string{"", "../escape", "/abs", "a/../b"}
	for _, c := range cases {
		if _, err := sanitizeKey(c); err == nil {
			t.Fatalf("expected error for key %q", c)
		}
	}
}

// TestFilesystem_ListMetaCorrupt covers error path when meta file invalid JSON.
func TestFilesystem_ListMetaCorrupt(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(dir)
	if err != nil {
		t.Fatalf("new fs: %v", err)
	}
	// create a data file and corrupt meta
	data := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(data, []byte("data"), 0o644); err != nil {
		t.Fatalf("write data: %v", err)
	}
	if err := os.WriteFile(data+".meta", []byte("{"), 0o644); err != nil { // invalid json
		t.Fatalf("write meta: %v", err)
	}
	if _, err := fs.List(context.Background(), ""); err == nil {
		t.Fatalf("expected list error on corrupt meta")
	}
}

// TestCloneMD ensures deep copy and nil handling.
func TestCloneMD(t *testing.T) {
	if cloneMD(nil) != nil { //nolint:goerr113 // simple test
		t.Fatalf("expected nil pass-through")
	}
	src := map[string]string{"a": "1"}
	cp := cloneMD(src)
	if cp["a"] != "1" || len(cp) != 1 {
		t.Fatalf("copy mismatch: %#v", cp)
	}
	src["a"] = "2"
	if cp["a"] != "1" { // ensure deep copy
		t.Fatalf("expected deep copy isolation")
	}
}

// TestWriteJSONMarshalError triggers marshal error by swapping jsonMarshal.
func TestWriteJSONMarshalError(t *testing.T) {
	old := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("marsh") }
	defer func() { jsonMarshal = old }()
	if err := writeJSON(filepath.Join(t.TempDir(), "x.meta"), struct{}{}); err == nil {
		t.Fatalf("expected marshal error")
	}
}

// TestReadMetaUnmarshalError triggers json unmarshal error.
func TestReadMetaUnmarshalError(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.meta")
	if err := os.WriteFile(f, []byte("{"), 0o644); err != nil { // invalid json
		t.Fatalf("write: %v", err)
	}
	if _, err := readMeta(f); err == nil {
		t.Fatalf("expected unmarshal error")
	}
}
