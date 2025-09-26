package datasets_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"colonycore/internal/adapters/datasets"
	"colonycore/internal/core"
	domain "colonycore/pkg/domain"
	"colonycore/plugins/frog"
)

type listResponse struct {
	Templates []core.DatasetTemplateDescriptor `json:"templates"`
}

type templateResponse struct {
	Template core.DatasetTemplateDescriptor `json:"template"`
}

type validateResponse struct {
	Valid  bool                         `json:"valid"`
	Errors []core.DatasetParameterError `json:"errors"`
}

type runResponse struct {
	Result struct {
		Rows []map[string]any `json:"rows"`
	} `json:"result"`
}

func setupHandler(t *testing.T) (*core.Service, *datasets.Handler, core.DatasetTemplateDescriptor) {
	t.Helper()
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	handler := datasets.NewHandler(svc)
	return svc, handler, meta.Datasets[0]
}

func TestHandlerListTemplates(t *testing.T) {
	_, handler, descriptor := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}

	var body listResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Templates) != 1 {
		t.Fatalf("expected one template")
	}
	if body.Templates[0].Slug != descriptor.Slug {
		t.Fatalf("unexpected slug: %s", body.Templates[0].Slug)
	}
}

func TestHandlerGetTemplate(t *testing.T) {
	_, handler, descriptor := setupHandler(t)

	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version
	req := httptest.NewRequest(http.MethodGet, url, nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	var body templateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Template.Slug != descriptor.Slug {
		t.Fatalf("unexpected slug: %s", body.Template.Slug)
	}
}

func TestHandlerValidateErrors(t *testing.T) {
	_, handler, descriptor := setupHandler(t)

	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/validate"
	body := bytes.NewBufferString(`{"parameters": {"unknown": "value"}}`)
	req := httptest.NewRequest(http.MethodPost, url, body)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	var validation validateResponse
	if err := json.NewDecoder(resp.Body).Decode(&validation); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if validation.Valid {
		t.Fatalf("expected validation to fail")
	}
	if len(validation.Errors) == 0 {
		t.Fatalf("expected validation errors")
	}
}

func TestHandlerRunJSON(t *testing.T) {
	svc, handler, descriptor := setupHandler(t)

	ctx := context.Background()
	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-HTTP", Title: "Dataset"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectID := project.ID
	if _, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog", Species: "Tree Frog", Stage: domain.StageAdult, ProjectID: &projectID}); err != nil {
		t.Fatalf("create organism: %v", err)
	}

	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/run"
	payload := `{"scope": {"project_ids": ["` + project.ID + `"]}, "parameters": {"include_retired": true}}`
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}

	var result runResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Result.Rows) != 1 {
		t.Fatalf("expected single row, got %d", len(result.Result.Rows))
	}
}

func TestHandlerRunCSV(t *testing.T) {
	svc, handler, descriptor := setupHandler(t)

	ctx := context.Background()
	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-CSV", Title: "Dataset"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectID := project.ID
	if _, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog", Species: "Tree Frog", Stage: domain.StageAdult, ProjectID: &projectID}); err != nil {
		t.Fatalf("create organism: %v", err)
	}

	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/run?format=csv"
	payload := `{"scope": {"project_ids": ["` + project.ID + `"]}, "parameters": {"include_retired": true}}`
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "text/csv" {
		t.Fatalf("unexpected content type: %s", got)
	}
	reader := csv.NewReader(resp.Body)
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	if len(rows) != 2 { // header + single row
		t.Fatalf("expected 2 csv rows, got %d", len(rows))
	}
}

func TestHandlerExportLifecycle(t *testing.T) {
	svc, handler, descriptor := setupHandler(t)
	store := datasets.NewMemoryObjectStore()
	audit := &datasets.MemoryAuditLog{}
	worker := datasets.NewWorker(svc, store, audit)
	handler.Exports = worker
	worker.Start()
	t.Cleanup(func() {
		_ = worker.Stop(context.Background())
	})

	ctx := context.Background()
	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-EXP", Title: "Export"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectID := project.ID
	if _, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog", Species: "Tree Frog", Stage: domain.StageAdult, ProjectID: &projectID}); err != nil {
		t.Fatalf("create organism: %v", err)
	}

	payload := fmt.Sprintf(`{"template":{"slug":"%s"},"scope":{"project_ids":["%s"],"requestor":"analyst@colonycore"},"parameters":{"include_retired":true},"formats":["json","csv","html","png","parquet"],"requested_by":"analyst@colonycore"}`, descriptor.Slug, project.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", bytes.NewBufferString(payload))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", resp.Code)
	}

	var created struct {
		Export datasets.ExportRecord `json:"export"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode export create: %v", err)
	}
	if created.Export.ID == "" {
		t.Fatalf("expected export id")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := handler.Exports.GetExport(created.Export.ID)
		if record.Status == datasets.ExportStatusSucceeded {
			break
		}
		if record.Status == datasets.ExportStatusFailed {
			t.Fatalf("export failed: %s", record.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for export completion (status=%s)", record.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports/"+created.Export.ID, nil)
	statusResp := httptest.NewRecorder()
	handler.ServeHTTP(statusResp, statusReq)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("unexpected status response: %d", statusResp.Code)
	}

	objects := store.Objects()
	if len(objects) == 0 {
		t.Fatalf("expected stored artifacts")
	}
}

func TestHandlerTemplateNotFound(t *testing.T) {
	_, handler, _ := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates/frog/unknown/1.0.0", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

func TestHandlerRunUnsupportedFormat(t *testing.T) {
	_, handler, descriptor := setupHandler(t)
	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/run?format=parquet"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"parameters":{}}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotAcceptable {
		t.Fatalf("expected 406 for unsupported format, got %d", resp.Code)
	}
}

func TestHandlerValidateInvalidJSON(t *testing.T) {
	_, handler, descriptor := setupHandler(t)
	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/validate"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString("{invalid"))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestHandlerExportCreateMissingTemplate(t *testing.T) {
	_, handler, _ := setupHandler(t)
	handler.Exports = datasets.NewWorker(nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", bytes.NewBufferString(`{}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing template, got %d", resp.Code)
	}
}

func TestHandlerExportGetNotFound(t *testing.T) {
	_, handler, _ := setupHandler(t)
	handler.Exports = datasets.NewWorker(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports/missing", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing export, got %d", resp.Code)
	}
}

func TestHandlerTemplateMethodNotAllowed(t *testing.T) {
	_, handler, descriptor := setupHandler(t)
	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version
	req := httptest.NewRequest(http.MethodPost, url, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.Code)
	}
}

func TestHandlerExportsMethodNotAllowed(t *testing.T) {
	_, handler, _ := setupHandler(t)
	handler.Exports = datasets.NewWorker(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", resp.Code)
	}
}

func TestHandlerRunParameterErrors(t *testing.T) {
	_, handler, descriptor := setupHandler(t)
	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/run"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"parameters":{"unknown":true}}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for parameter validation, got %d", resp.Code)
	}
}

func TestHandlerExportCreateUnsupportedFormat(t *testing.T) {
	_, handler, descriptor := setupHandler(t)
	handler.Exports = datasets.NewWorker(nil, nil, nil)
	payload := fmt.Sprintf(`{"template":{"plugin":"%s","key":"%s","version":"%s"},"formats":["xml"]}`, descriptor.Plugin, descriptor.Key, descriptor.Version)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", bytes.NewBufferString(payload))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported export format, got %d", resp.Code)
	}
}

func TestHandlerServeHTTPUnknownPath(t *testing.T) {
	_, handler, _ := setupHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/ping", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", resp.Code)
	}
}

func TestHandlerExportsPathMethodNotAllowed(t *testing.T) {
	_, handler, _ := setupHandler(t)
	handler.Exports = datasets.NewWorker(nil, nil, nil)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/datasets/exports/identifier", bytes.NewBufferString(`{}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for exports PUT, got %d", resp.Code)
	}
}
