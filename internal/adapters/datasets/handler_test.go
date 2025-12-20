package datasets

import (
	"bytes"
	"colonycore/internal/adapters/testutil"
	"colonycore/internal/core"
	"colonycore/internal/entitymodel"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Basic smoke test ensuring handler lists templates after plugin install.
func TestHandlerListTemplates_Smoke(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := testutil.InstallFrogPlugin(svc); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
}

func TestHandlerServesEntityModelOpenAPI(t *testing.T) {
	h := NewHandler(core.NewInMemoryService(core.NewDefaultRulesEngine()))
	// Cover the fallback path for nil handler.
	h.EntityModel = nil

	meta := entitymodel.MetadataInfo()

	req := httptest.NewRequest(http.MethodGet, "/admin/entity-model/openapi", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/yaml" {
		t.Fatalf("unexpected content type %q", got)
	}
	if got := rec.Header().Get("X-Entity-Model-Version"); got != meta.Version {
		t.Fatalf("unexpected version header %q", got)
	}
	if got := rec.Header().Get("X-Entity-Model-Status"); got != meta.Status {
		t.Fatalf("unexpected status header %q", got)
	}
	if got := rec.Header().Get("X-Entity-Model-Source"); got != meta.Source {
		t.Fatalf("unexpected source header %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), entitymodel.OpenAPISpec()) {
		t.Fatalf("openapi body mismatch")
	}
}
