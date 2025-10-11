package datasets

import (
	"bytes"
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

// Additional handler tests migrated from external package variant.
func TestHandlerListTemplatesAndNewHandler(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	plugin := testDatasetPlugin{dataset: datasetapi.Template{Key: "list", Version: "1.0.0", Title: "List", Description: "list test", Dialect: dialectProvider.SQL(), Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "value", Type: "string"}}, OutputFormats: []datasetapi.Format{formatProvider.JSON()}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
	}}}
	if _, err := svc.InstallPlugin(plugin); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	h := NewHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("list")) {
		t.Fatalf("expected body to mention template key, got %s", rec.Body.String())
	}
}

func TestHandlerValidateSuccess(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	template := datasetapi.Template{Key: "validate", Version: "1.0.0", Title: "Validate", Description: "validation", Dialect: dialectProvider.SQL(), Query: "SELECT 1", Parameters: []datasetapi.Parameter{{Name: "stage", Type: "string", Enum: []string{"adult", "larva"}}}, Columns: []datasetapi.Column{{Name: "value", Type: "string"}}, OutputFormats: []datasetapi.Format{formatProvider.JSON()}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
	}}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	d := svc.DatasetTemplates()[0]
	h := NewHandler(svc)
	url := "/api/v1/datasets/templates/" + d.Plugin + "/" + d.Key + "/" + d.Version + "/validate"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"parameters":{"stage":"adult"}}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("\"valid\":true")) {
		t.Fatalf("expected valid true, got %s", rec.Body.String())
	}
}

func TestNegotiateFormatCSVHeader(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	template := datasetapi.Template{Key: "csv", Version: "1.0.0", Title: "CSV", Description: "csv", Dialect: dialectProvider.SQL(), Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "value", Type: "string"}}, OutputFormats: []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{Rows: []datasetapi.Row{{"value": "ok"}}, Format: formatProvider.JSON()}, nil
		}, nil
	}}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	d := svc.DatasetTemplates()[0]
	h := NewHandler(svc)
	runURL := "/api/v1/datasets/templates/" + d.Plugin + "/" + d.Key + "/" + d.Version + "/run"
	req := httptest.NewRequest(http.MethodPost, runURL, bytes.NewBufferString(`{"parameters":{}}`))
	req.Header.Set("Accept", "text/csv")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected csv content type, got %s", ct)
	}
	reader := csv.NewReader(bytes.NewBuffer(rec.Body.Bytes()))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) == 0 || rows[0][0] != "value" {
		t.Fatalf("unexpected csv header: %+v", rows)
	}
	badReq := httptest.NewRequest(http.MethodPost, runURL+"?format=parquet", bytes.NewBufferString(`{"parameters":{}}`))
	badRec := httptest.NewRecorder()
	h.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusNotAcceptable {
		t.Fatalf("expected 406, got %d", badRec.Code)
	}
}

func TestRunCSVUsesDescriptorColumns(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	template := datasetapi.Template{Key: "empty", Version: "1.0.0", Title: "EmptySchema", Description: "empty schema", Dialect: dialectProvider.SQL(), Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "col", Type: "string"}}, OutputFormats: []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{Rows: []datasetapi.Row{{"col": "alpha"}}, Format: formatProvider.JSON()}, nil
		}, nil
	}}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	d := svc.DatasetTemplates()[0]
	h := NewHandler(svc)
	runURL := "/api/v1/datasets/templates/" + d.Plugin + "/" + d.Key + "/" + d.Version + "/run?format=csv"
	req := httptest.NewRequest(http.MethodPost, runURL, bytes.NewBufferString(`{"parameters":{}}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	reader := csv.NewReader(bytes.NewBuffer(rec.Body.Bytes()))
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) < 2 || rows[0][0] != "col" {
		t.Fatalf("expected header fallback, got %+v", rows)
	}
}

func TestMemoryAuditLogEntries(t *testing.T) {
	log := &MemoryAuditLog{}
	if len(log.Entries()) != 0 {
		t.Fatalf("expected no entries initially")
	}
	log.Record(context.Background(), AuditEntry{Actor: "tester", Action: "unit"})
	entries := log.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected one entry")
	}
	entries[0].Actor = "mutated"
	again := log.Entries()
	if again[0].Actor != "tester" {
		t.Fatalf("expected defensive copy, got %+v", again[0])
	}
}

func TestServeHTTPExportsLifecycle(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	template := datasetapi.Template{Key: "lifecycle", Version: "1.0.0", Title: "Lifecycle", Description: "export lifecycle", Dialect: dialectProvider.SQL(), Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "value", Type: "string"}}, OutputFormats: []datasetapi.Format{formatProvider.JSON()}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{Rows: []datasetapi.Row{{"value": "ok"}}, Format: formatProvider.JSON()}, nil
		}, nil
	}}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	worker := NewWorker(svc, NewMemoryObjectStore(), &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() { _ = worker.Stop(context.Background()) })
	h := NewHandler(svc)
	h.Exports = worker
	d := svc.DatasetTemplates()[0]
	createPayload := bytes.NewBufferString(`{"template":{"plugin":"` + d.Plugin + `","key":"` + d.Key + `","version":"` + d.Version + `"},"formats":["json"],"requested_by":"tester"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", createPayload)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	body := rec.Body.String()
	idx := strings.Index(body, `"id":"`)
	if idx == -1 {
		t.Fatalf("expected id in response: %s", body)
	}
	idx += len(`"id":"`)
	end := idx
	for end < len(body) && body[end] != '"' {
		end++
	}
	id := body[idx:end]
	deadline := time.Now().Add(2 * time.Second)
	for {
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports/"+id, nil)
		getRec := httptest.NewRecorder()
		h.ServeHTTP(getRec, getReq)
		if getRec.Code == http.StatusOK && strings.Contains(getRec.Body.String(), "\"status\":\"succeeded\"") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not succeed in time; last=%d %s", getRec.Code, getRec.Body.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
