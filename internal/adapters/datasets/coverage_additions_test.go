package datasets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	"colonycore/plugins/frog"
)

// (Helper types are defined in exporter_worker_test.go; reuse them here.)

func TestWorkerEnqueueErrors(t *testing.T) {
	w := NewWorker(nil, nil, nil)
	if _, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: "frog/list@1.0.0"}); err == nil {
		t.Fatalf("expected error when catalog nil")
	}
}

func TestWorkerEnqueueBlankSlug(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	w := NewWorker(svc, nil, nil)
	if _, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: ""}); err == nil {
		t.Fatalf("expected error for blank slug")
	}
}

func TestWorkerEnqueueDuplicateFormatsAndQueueFull(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()

	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	slug := meta.Datasets[0].Slug
	w := NewWorker(svc, nil, nil) // do NOT start worker so queue fills

	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug, Formats: []datasetapi.Format{formatProvider.JSON(), formatProvider.JSON(), formatProvider.CSV()}})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// expect de-duplicated formats preserving first appearances order
	if len(rec.Formats) != 2 || rec.Formats[0] != formatProvider.JSON() || rec.Formats[1] != formatProvider.CSV() {
		t.Fatalf("unexpected formats after dedup: %#v", rec.Formats)
	}

	// Fill queue completely (already has one)
	capacity := cap(w.queue)
	for i := 1; i < capacity; i++ { // one already used
		if _, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug}); err != nil {
			t.Fatalf("unexpected enqueue error while filling queue: %v", err)
		}
	}
	if _, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug}); err == nil {
		t.Fatalf("expected queue full error")
	}
}

func TestWorkerGetExportUnknownAndActorTemplateScopeMissing(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	w := NewWorker(svc, nil, nil)
	if _, ok := w.GetExport("missing"); ok {
		t.Fatalf("expected missing export to return ok=false")
	}
	if got := w.actorFor("missing"); got != "" {
		t.Fatalf("expected empty actor, got %q", got)
	}
	if got := w.templateFor("missing"); got != "" {
		t.Fatalf("expected empty template, got %q", got)
	}
	if scope := w.scopeFor("missing"); len(scope.ProjectIDs)+len(scope.ProtocolIDs) != 0 || scope.Requestor != "" {
		t.Fatalf("expected empty scope, got %#v", scope)
	}
}

func TestHandlerServeHTTPNoCatalog(t *testing.T) {
	h := &Handler{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestHandlerRunAcceptHeaderNegotiation(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := meta.Datasets[0]
	h := NewHandler(svc)

	ctx := context.Background()
	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-ACPT", Title: "Negotiate"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectID := project.ID
	if _, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog", Species: "Tree Frog", Stage: domain.StageAdult, ProjectID: &projectID}); err != nil {
		t.Fatalf("create organism: %v", err)
	}

	url := fmt.Sprintf("/api/v1/datasets/templates/%s/%s/%s/run", descriptor.Plugin, descriptor.Key, descriptor.Version)
	payload := fmt.Sprintf(`{"scope":{"project_ids":["%s"]},"parameters":{"include_retired":true}}`, project.ID)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
	req.Header.Set("Accept", "text/csv") // triggers Accept negotiation path
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("expected csv content type (Accept negotiation), got %s", ct)
	}
}

func TestHandlerRunInternalError(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	dialectProvider := datasetapi.GetDialectProvider()
	failing := core.DatasetTemplate{
		Plugin: "frog",
		Template: datasetapi.Template{
			Key:           "failing2",
			Version:       "1.0.0",
			Title:         "Failing2",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Columns:       []datasetapi.Column{{Name: "v", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}
	failing.RunnerForTests(func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
		return core.DatasetRunResult{}, fmt.Errorf("boom")
	})
	fc := fakeCatalog{tpl: core.DatasetTemplateRuntimeForTests(failing)}
	h := NewHandler(fc)
	url := fmt.Sprintf("/api/v1/datasets/templates/%s/%s/%s/run", failing.Plugin, failing.Key, failing.Version)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"parameters":{}}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// (Unsupported materialize format already covered in exporter_worker_test.go)

func TestWorkerStopContextTimeoutBranch(_ *testing.T) {
	w := NewWorker(nil, nil, nil)
	w.Start()
	// Expired context to attempt selecting ctx.Done path.
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	// Cancel worker to allow it to finish quickly.
	_ = w.Stop(ctx) // We don't assert error here; goal is branch execution.
}

func TestHandlerExportCreateTemplateSlugFallback(t *testing.T) {
	// Provide export create path where slug components are used instead of direct slug.
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := meta.Datasets[0]
	w := NewWorker(svc, nil, nil)
	h := NewHandler(svc)
	h.Exports = w
	payload := fmt.Sprintf(`{"template":{"plugin":"%s","key":"%s","version":"%s"},"formats":["json"],"requested_by":"user@colony"}`, descriptor.Plugin, descriptor.Key, descriptor.Version)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}
	var body struct {
		Export ExportRecord `json:"export"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Export.Template.Slug == "" {
		t.Fatalf("expected export record with template descriptor")
	}
}

func TestHandlerExportsBranches(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	h := NewHandler(svc)
	h.Exports = NewWorker(svc, nil, nil)

	// Method not allowed (GET on collection path expecting POST)
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports", nil)
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr1.Code)
	}

	// Missing ID (path with trailing slash but no id) resolves to collection path (trim suffix) -> 405
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports/", nil)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for empty id path, got %d", rr2.Code)
	}

	// Unknown id
	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/exports/does-not-exist", nil)
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing export, got %d", rr3.Code)
	}
}
