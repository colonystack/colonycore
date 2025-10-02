package datasets

import (
	"colonycore/internal/core"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// NOTE: reuse fakeCatalog type defined in exporter_worker_test.go (with field tpl)

// buildSimpleTemplate returns a minimal bound template.
func buildSimpleTemplate() core.DatasetTemplate {
	tmpl := core.DatasetTemplate{Key: "k", Version: "1", Title: "T", Dialect: core.DatasetDialectSQL, Query: "SELECT 1", Columns: []core.DatasetColumn{{Name: "v", Type: "string"}}, OutputFormats: []core.DatasetFormat{core.FormatJSON}}
	core.BindTemplateForTests(&tmpl, func(_ context.Context, _ core.DatasetRunRequest) (core.DatasetRunResult, error) {
		return core.DatasetRunResult{Rows: []map[string]any{{"v": "x"}}, Schema: tmpl.Columns, GeneratedAt: time.Unix(0, 0).UTC()}, nil
	})
	return tmpl
}

func TestHandleTemplateMissingSegments(t *testing.T) {
	h := NewHandler(fakeCatalog{tpl: buildSimpleTemplate()})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/frog", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleTemplateUnknownSlug(t *testing.T) {
	h := NewHandler(fakeCatalog{tpl: buildSimpleTemplate()})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/frog/unknown/2", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown slug, got %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] == "" {
		t.Fatalf("expected error message in body")
	}
}
