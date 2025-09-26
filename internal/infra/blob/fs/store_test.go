package fs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"colonycore/internal/blob/core"
)

func newTempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return store
}

func TestStore_PutGetHeadListDelete(t *testing.T) { //nolint:cyclop
	ctx := context.Background()
	store := newTempStore(t)
	info, err := store.Put(ctx, "alpha/test.txt", bytes.NewReader([]byte("hello")), core.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"k": "v"}})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if info.Key != "alpha/test.txt" || info.Size != 5 {
		t.Fatalf("unexpected info %+v", info)
	}
	if _, err := store.Put(ctx, "alpha/test.txt", bytes.NewReader([]byte("x")), core.PutOptions{}); err == nil {
		t.Fatalf("expected duplicate failure")
	}
	h, err := store.Head(ctx, "alpha/test.txt")
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	g, rc, err := store.Get(ctx, "alpha/test.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	b, _ := io.ReadAll(rc)
	if err := rc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if string(b) != "hello" || g.ETag != h.ETag {
		t.Fatalf("unexpected get artifacts")
	}
	list, err := store.List(ctx, "alpha/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Key != "alpha/test.txt" {
		t.Fatalf("unexpected list %+v", list)
	}
	url, err := store.PresignURL(ctx, "alpha/test.txt", core.SignedURLOptions{})
	if err != nil || url == "" {
		t.Fatalf("presign url: %v %s", err, url)
	}
	ok, err := store.Delete(ctx, "alpha/test.txt")
	if err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	ok, err = store.Delete(ctx, "alpha/test.txt")
	if err != nil || ok {
		t.Fatalf("second delete should be false")
	}
}

func TestStore_PathTraversal(t *testing.T) {
	ctx := context.Background()
	store := newTempStore(t)
	if _, err := store.Put(ctx, "../escape.txt", bytes.NewReader([]byte("x")), core.PutOptions{}); err == nil {
		t.Fatalf("expected traversal error")
	}
	if _, err := store.Put(ctx, "/abs.txt", bytes.NewReader([]byte("x")), core.PutOptions{}); err == nil {
		t.Fatalf("expected absolute error")
	}
}

func TestStore_MetadataPersistence(t *testing.T) {
	ctx := context.Background()
	store := newTempStore(t)
	if _, err := store.Put(ctx, "meta/data.bin", bytes.NewReader([]byte("abc")), core.PutOptions{ContentType: "application/octet-stream", Metadata: map[string]string{"a": "1"}}); err != nil {
		t.Fatalf("put: %v", err)
	}
	dataPath, metaPath, _ := store.pathFor("meta/data.bin")
	if _, err := os.Stat(dataPath); err != nil {
		t.Fatalf("expected data path: %v", err)
	}
	b, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if !bytes.Contains(b, []byte("application/octet-stream")) {
		t.Fatalf("meta missing content type")
	}
	if filepath.Ext(metaPath) != ".meta" {
		t.Fatalf("meta path extension mismatch")
	}
}

type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) { return 0, errors.New("boom") }

func TestStore_PutDuplicateAndErrorBranches(t *testing.T) {
	store := newTempStore(t)
	if _, err := store.Put(context.Background(), "k1.txt", bytes.NewReader([]byte("hi")), core.PutOptions{ContentType: "text/plain", Metadata: map[string]string{"a": "1"}}); err != nil {
		t.Fatalf("put1: %v", err)
	}
	if _, err := store.Put(context.Background(), "k1.txt", bytes.NewReader([]byte("again")), core.PutOptions{}); err == nil {
		t.Fatalf("expected duplicate put error")
	}
	if _, err := store.Put(context.Background(), "bad.bin", errorReader{}, core.PutOptions{}); err == nil {
		t.Fatalf("expected copy error")
	}
}

func TestStore_HeadGetDeleteAndList(t *testing.T) { //nolint:cyclop
	dir := t.TempDir()
	store, _ := New(dir)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		k := filepath.Join("folder", "f"+strconv.Itoa(i)+".txt")
		if _, err := store.Put(ctx, k, bytes.NewReader([]byte("data")), core.PutOptions{}); err != nil {
			t.Fatalf("put %s: %v", k, err)
		}
	}
	if _, err := store.Head(ctx, "folder/f0.txt"); err != nil {
		t.Fatalf("head: %v", err)
	}
	if _, rc, err := store.Get(ctx, "folder/f1.txt"); err != nil {
		t.Fatalf("get: %v", err)
	} else {
		_ = rc.Close()
	}
	if ok, err := store.Delete(ctx, "folder/f2.txt"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	if ok, err := store.Delete(ctx, "folder/missing.txt"); err != nil || ok {
		t.Fatalf("expected delete false")
	}
	list, err := store.List(ctx, "folder/")
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	_, metaPath, _ := store.pathFor("folder/f0.txt")
	if err := os.Remove(metaPath); err != nil {
		t.Fatalf("rm meta: %v", err)
	}
	if _, _, err := store.Get(ctx, "folder/f0.txt"); err == nil {
		t.Fatalf("expected get meta error")
	}
	if _, err := store.Head(ctx, "folder/f0.txt"); err == nil {
		t.Fatalf("expected head meta error")
	}
}

func TestStore_PresignVariantsAndListOrder(t *testing.T) {
	ctx := context.Background()
	store := newTempStore(t)
	if _, err := store.Put(ctx, "a/1.txt", bytes.NewReader([]byte("a1")), core.PutOptions{}); err != nil {
		t.Fatalf("put1: %v", err)
	}
	if _, err := store.Put(ctx, "b/2.txt", bytes.NewReader([]byte("b2")), core.PutOptions{}); err != nil {
		t.Fatalf("put2: %v", err)
	}
	if url, err := store.PresignURL(ctx, "a/1.txt", core.SignedURLOptions{Method: "get"}); err != nil || url == "" {
		t.Fatalf("presign lower: %v %s", err, url)
	}
	if _, err := store.PresignURL(ctx, "a/1.txt", core.SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected unsupported for PUT")
	}
	list, err := store.List(ctx, "")
	if err != nil || len(list) != 2 {
		t.Fatalf("list root: %v %v", err, list)
	}
	if list[0].Key > list[1].Key {
		t.Fatalf("expected sorted order: %+v", list)
	}
}

func TestStore_PresignUnsupported(t *testing.T) {
	store := newTempStore(t)
	if _, err := store.PresignURL(context.Background(), "some", core.SignedURLOptions{Method: "PUT"}); err == nil {
		t.Fatalf("expected unsupported error")
	}
}

// Helpers ------------------------------------------------------------------------------------------------

func TestSanitizeKeyErrors(t *testing.T) {
	cases := []string{"", "../escape", "/abs", "a/../b"}
	for _, c := range cases {
		if _, err := sanitizeKey(c); err == nil {
			t.Fatalf("expected error for key %q", c)
		}
	}
}

func TestListMetaCorrupt(t *testing.T) {
	dir := t.TempDir()
	store, err := New(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	data := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(data, []byte("data"), 0o644); err != nil {
		t.Fatalf("write data: %v", err)
	}
	if err := os.WriteFile(data+".meta", []byte("{"), 0o644); err != nil {
		t.Fatalf("write meta: %v", err)
	}
	if _, err := store.List(context.Background(), ""); err == nil {
		t.Fatalf("expected list error on corrupt meta")
	}
}

func TestCloneMetadata(t *testing.T) {
	if cloneMetadata(nil) != nil {
		t.Fatalf("expected nil pass-through")
	}
	src := map[string]string{"a": "1"}
	cp := cloneMetadata(src)
	if cp["a"] != "1" || len(cp) != 1 {
		t.Fatalf("copy mismatch: %#v", cp)
	}
	src["a"] = "2"
	if cp["a"] != "1" {
		t.Fatalf("expected deep copy isolation")
	}
}

func TestWriteJSONMarshalError(t *testing.T) {
	old := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("marsh") }
	defer func() { jsonMarshal = old }()
	if err := writeJSON(filepath.Join(t.TempDir(), "x.meta"), struct{}{}); err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestReadMetaUnmarshalError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "bad.meta")
	if err := os.WriteFile(file, []byte("{"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := readMeta(file); err == nil {
		t.Fatalf("expected unmarshal error")
	}
}

func TestNewRejectsFileRoot(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := New(filePath); err == nil {
		t.Fatalf("expected error when root is file")
	}
}

func TestStoreLocalURLStable(t *testing.T) {
	store := &Store{root: t.TempDir()}
	if url := store.localURL("path/to.obj"); url != "http://local.blob/path/to.obj" {
		t.Fatalf("unexpected url: %s", url)
	}
}

func TestStoreMetadataTimestampsUTC(t *testing.T) {
	ctx := context.Background()
	store := newTempStore(t)
	info, err := store.Put(ctx, "time/test", bytes.NewReader([]byte("abc")), core.PutOptions{})
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if !info.LastModified.Equal(info.LastModified.UTC()) {
		t.Fatalf("expected UTC timestamp")
	}
	if _, err := store.Head(ctx, "time/test"); err != nil {
		t.Fatalf("head: %v", err)
	}
	if list, err := store.List(ctx, "time/"); err != nil || len(list) != 1 {
		t.Fatalf("list: %v %d", err, len(list))
	}
}
