package datasets

import (
	"colonycore/internal/adapters/testutil"
	"colonycore/internal/core"
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
