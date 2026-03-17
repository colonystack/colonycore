package datasets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

// NOTE: reuse fakeCatalog type defined in exporter_worker_test.go (with field tpl)

// buildSimpleTemplate returns a minimal bound template.
func buildSimpleTemplate() core.DatasetTemplate {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	tmpl := core.DatasetTemplate{
		Template: datasetapi.Template{
			Key:           "k",
			Version:       "1",
			Title:         "T",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Columns:       []datasetapi.Column{{Name: "v", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}
	core.BindTemplateForTests(&tmpl, func(_ context.Context, _ datasetapi.RunRequest) (datasetapi.RunResult, error) {
		return datasetapi.RunResult{Rows: []datasetapi.Row{{"v": "x"}}, Schema: tmpl.Columns, GeneratedAt: time.Unix(0, 0).UTC()}, nil
	})
	return tmpl
}

func TestHandleTemplateMissingSegments(t *testing.T) {
	h := NewHandler(fakeCatalog{tpl: core.DatasetTemplateRuntimeForTests(buildSimpleTemplate())})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/frog", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleTemplateUnknownSlug(t *testing.T) {
	h := NewHandler(fakeCatalog{tpl: core.DatasetTemplateRuntimeForTests(buildSimpleTemplate())})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/frog/unknown/2", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown slug, got %d", rr.Code)
	}
	var problem problemDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Type != problemTypeBlank {
		t.Fatalf("expected problem type %q, got %q", problemTypeBlank, problem.Type)
	}
	if problem.Title != http.StatusText(http.StatusNotFound) {
		t.Fatalf("expected title %q, got %q", http.StatusText(http.StatusNotFound), problem.Title)
	}
	if problem.Status != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, problem.Status)
	}
	if problem.Detail == "" {
		t.Fatalf("expected problem detail in body")
	}
}
