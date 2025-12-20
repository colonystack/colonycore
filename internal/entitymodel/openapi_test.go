package entitymodel

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenAPISpecReturnsCopy(t *testing.T) {
	expected := readEmbeddedSpec(t)
	spec := OpenAPISpec()

	if len(spec) == 0 {
		t.Fatal("expected non-empty OpenAPI spec")
	}
	if !bytes.Equal(spec, expected) {
		t.Fatalf("OpenAPISpec does not match embedded contents")
	}

	spec[0] ^= 0xFF
	next := OpenAPISpec()
	if bytes.Equal(spec, next) {
		t.Fatalf("OpenAPISpec did not return a defensive copy")
	}
	if !bytes.Equal(next, expected) {
		t.Fatalf("OpenAPISpec mutation leaked into source")
	}
}

func TestNewOpenAPIHandlerServesEmbeddedSpec(t *testing.T) {
	expected := readEmbeddedSpec(t)

	req := httptest.NewRequest(http.MethodGet, "/openapi", nil)
	rec := httptest.NewRecorder()

	NewOpenAPIHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/yaml" {
		t.Fatalf("expected Content-Type application/yaml, got %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("handler body does not match embedded spec")
	}
}

func readEmbeddedSpec(t *testing.T) []byte {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot determine caller")
	}

	path := filepath.Join(filepath.Dir(filename), "..", "..", "docs", "schema", "openapi", "entity-model.yaml")
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read embedded spec: %v", err)
	}
	return data
}
