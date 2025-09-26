package dataset

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

func TestHandleTemplateMissingSegments(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	handler := &Handler{Catalog: svc}
	req := httptest.NewRequest(http.MethodGet, "/ignored", nil)
	rec := httptest.NewRecorder()
	handler.handleTemplate(rec, req, "frog")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleTemplateMethodNotAllowedInternal(t *testing.T) {
	template := datasetapi.Template{
		Key:           "demo",
		Version:       "1.0.0",
		Title:         "Demo",
		Description:   "demo",
		Dialect:       datasetapi.DialectSQL,
		Query:         "SELECT 1",
		Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
		OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := svc.DatasetTemplates()[0]
	handler := &Handler{Catalog: svc}
	remainder := descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version
	req := httptest.NewRequest(http.MethodPost, "/ignored", nil)
	rec := httptest.NewRecorder()
	handler.handleTemplate(rec, req, remainder)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestServeHTTPMissingCatalog(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when catalog missing, got %d", rec.Code)
	}
}

func TestHandleExportsInvalidPath(t *testing.T) {
	handler := &Handler{Exports: NewWorker(nil, nil, nil)}
	req := httptest.NewRequest(http.MethodGet, "/ignored", nil)
	rec := httptest.NewRecorder()
	handler.handleExports(rec, req, "/api/v1/datasets/exports/foo/bar")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleRunRunnerError(t *testing.T) {
	template := datasetapi.Template{
		Key:           "error",
		Version:       "1.0.0",
		Title:         "Error",
		Description:   "demo",
		Dialect:       datasetapi.DialectSQL,
		Query:         "SELECT 1",
		Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
		OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{}, fmt.Errorf("failure")
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := svc.DatasetTemplates()[0]
	bound, ok := svc.ResolveDatasetTemplate(descriptor.Slug)
	if !ok {
		t.Fatalf("expected dataset template to resolve")
	}
	handler := &Handler{Catalog: svc}
	req := httptest.NewRequest(http.MethodPost, "/ignored", bytes.NewBufferString(`{"parameters":{}}`))
	rec := httptest.NewRecorder()
	handler.handleRun(rec, req, bound)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestHandleExportCreateWithPluginFields(t *testing.T) {
	template := datasetapi.Template{
		Key:           "create",
		Version:       "1.0.0",
		Title:         "Create",
		Description:   "demo",
		Dialect:       datasetapi.DialectSQL,
		Query:         "SELECT 1",
		Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
		OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	worker := NewWorker(svc, NewMemoryObjectStore(), &MemoryAuditLog{})
	handler := &Handler{Catalog: svc, Exports: worker}
	descriptor := svc.DatasetTemplates()[0]
	payload := bytes.NewBufferString(fmt.Sprintf(`{"template":{"plugin":"%s","key":"%s","version":"%s"},"formats":["json"]}`, descriptor.Plugin, descriptor.Key, descriptor.Version))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", payload)
	rec := httptest.NewRecorder()
	handler.handleExportCreate(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
}
