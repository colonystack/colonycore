package datasets

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

func TestHandlerExportsVariants(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	hNoSched := NewHandler(svc)
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports", nil)
	getRec := httptest.NewRecorder()
	hNoSched.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", getRec.Code)
	}
	worker := NewWorker(svc, NewMemoryObjectStore(), &MemoryAuditLog{})
	h := NewHandler(svc)
	h.Exports = worker
	getRec2 := httptest.NewRecorder()
	h.ServeHTTP(getRec2, getReq)
	if getRec2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", getRec2.Code)
	}
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: datasetapi.Template{Key: "expv", Version: "1.0.0", Title: "E", Description: "E", Dialect: datasetapi.DialectSQL, Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "v", Type: "string"}}, OutputFormats: []datasetapi.Format{datasetapi.FormatJSON}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
	}}}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	slug := svc.DatasetTemplates()[0].Slug
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", bytes.NewBufferString(`{"template":{"slug":"`+slug+`"},"formats":["yaml"]}`))
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", postRec.Code)
	}
}

func TestTemplateVariants(t *testing.T) {
	template := datasetapi.Template{Key: "variants", Version: "1.0.0", Title: "Variants", Description: "variants", Dialect: datasetapi.DialectSQL, Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "value", Type: "string"}}, OutputFormats: []datasetapi.Format{datasetapi.FormatJSON}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{Rows: []datasetapi.Row{{"value": "ok"}}, Format: datasetapi.FormatJSON}, nil
		}, nil
	}}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	h := NewHandler(svc)
	d := svc.DatasetTemplates()[0]
	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/run", bytes.NewBufferString(`{"parameters":{}}`))
	runRec := httptest.NewRecorder()
	h.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected 200 run, got %d", runRec.Code)
	}
	valReq := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/validate", nil)
	valRec := httptest.NewRecorder()
	h.ServeHTTP(valRec, valReq)
	if valRec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", valRec.Code)
	}
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/unknown", nil)
	badRec := httptest.NewRecorder()
	h.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", badRec.Code)
	}
	nfReq := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/frog/missing/1.0.0/run", bytes.NewBufferString(`{"parameters":{}}`))
	nfRec := httptest.NewRecorder()
	h.ServeHTTP(nfRec, nfReq)
	if nfRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 missing template, got %d", nfRec.Code)
	}
}
