package datasets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

func TestHandlerExportProgressLifecycle(t *testing.T) {
	runtime, runStarted, releaseRun := newBlockedExportRuntime()
	catalog := fakeCatalog{tpl: runtime}
	worker := NewWorker(catalog, NewMemoryObjectStore(), &MemoryAuditLog{})
	worker.Start()
	defer func() { _ = worker.Stop(context.Background()) }()

	handler := &Handler{Catalog: catalog, Exports: worker}

	req := httptest.NewRequest(
		http.MethodPost,
		datasetExportsPath,
		strings.NewReader(`{"template":{"slug":"`+runtime.Descriptor().Slug+`"},"formats":["json"],"requested_by":"tester"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"progress_pct":0`)) {
		t.Fatalf("expected queued progress field in create response: %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"eta_seconds":null`)) {
		t.Fatalf("expected queued eta_seconds field in create response: %s", rec.Body.String())
	}

	var createResp struct {
		Export ExportRecord `json:"export"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Export.ProgressState != ExportProgressStateQueued {
		t.Fatalf("expected queued progress state, got %q", createResp.Export.ProgressState)
	}
	if createResp.Export.ArtifactReadiness != ExportArtifactReadinessPending {
		t.Fatalf("expected pending artifact readiness, got %q", createResp.Export.ArtifactReadiness)
	}
	if createResp.Export.ETASeconds != nil {
		t.Fatalf("expected nil eta for queued export, got %v", *createResp.Export.ETASeconds)
	}

	select {
	case <-runStarted:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for run to start")
	}

	waitForExportRecord(t, worker, createResp.Export.ID, 2*time.Second, func(record ExportRecord) bool {
		return record.Status == ExportStatusRunning && record.ProgressState == ExportProgressStateExecutingTemplate
	})

	getReq := httptest.NewRequest(http.MethodGet, datasetExportsPath+"/"+createResp.Export.ID, nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"progress_state":"executing_template"`)) {
		t.Fatalf("expected running progress state in status response: %s", getRec.Body.String())
	}
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"artifact_readiness":"pending"`)) {
		t.Fatalf("expected pending artifact readiness in status response: %s", getRec.Body.String())
	}

	var runningResp struct {
		Export ExportRecord `json:"export"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &runningResp); err != nil {
		t.Fatalf("decode running response: %v", err)
	}
	if runningResp.Export.ProgressPct != exportProgressExecutePct {
		t.Fatalf("expected running progress %d, got %d", exportProgressExecutePct, runningResp.Export.ProgressPct)
	}
	if runningResp.Export.ETASeconds == nil {
		t.Fatalf("expected eta while export is running")
	}

	close(releaseRun)

	waitForExportRecord(t, worker, createResp.Export.ID, 2*time.Second, func(record ExportRecord) bool {
		return record.Status == ExportStatusSucceeded
	})

	finalReq := httptest.NewRequest(http.MethodGet, datasetExportsPath+"/"+createResp.Export.ID, nil)
	finalRec := httptest.NewRecorder()
	handler.ServeHTTP(finalRec, finalReq)

	if finalRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", finalRec.Code, finalRec.Body.String())
	}
	if !bytes.Contains(finalRec.Body.Bytes(), []byte(`"progress_pct":100`)) {
		t.Fatalf("expected completed progress field in status response: %s", finalRec.Body.String())
	}
	if !bytes.Contains(finalRec.Body.Bytes(), []byte(`"artifact_readiness":"ready"`)) {
		t.Fatalf("expected ready artifact readiness in status response: %s", finalRec.Body.String())
	}

	var completeResp struct {
		Export ExportRecord `json:"export"`
	}
	if err := json.Unmarshal(finalRec.Body.Bytes(), &completeResp); err != nil {
		t.Fatalf("decode completed response: %v", err)
	}
	if completeResp.Export.ProgressState != ExportProgressStateCompleted {
		t.Fatalf("expected completed progress state, got %q", completeResp.Export.ProgressState)
	}
	if completeResp.Export.ETASeconds != nil {
		t.Fatalf("expected nil eta for completed export, got %v", *completeResp.Export.ETASeconds)
	}
}

func TestWorkerFailedExportReportsPartialArtifactReadiness(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	runtime := newStubRuntime(
		"frog",
		"partial",
		"1",
		"Partial",
		"partial dataset",
		[]datasetapi.Column{{Name: "value", Type: "integer"}},
		[]datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()},
	)
	runtime.runFn = func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
		return datasetapi.RunResult{
			Schema:      append([]datasetapi.Column(nil), runtime.desc.Columns...),
			Rows:        []datasetapi.Row{{"value": 7}},
			GeneratedAt: time.Unix(0, 0).UTC(),
			Format:      formatProvider.JSON(),
		}, nil, nil
	}

	worker := NewWorker(fakeCatalog{tpl: runtime}, &failAfterNStore{
		backing: NewMemoryObjectStore(),
		failAt:  2,
	}, nil)
	worker.Start()
	defer func() { _ = worker.Stop(context.Background()) }()

	rec, err := worker.EnqueueExport(context.Background(), ExportInput{
		TemplateSlug: runtime.Descriptor().Slug,
		Formats:      []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()},
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	failed := waitForExportRecord(t, worker, rec.ID, 2*time.Second, func(record ExportRecord) bool {
		return record.Status == ExportStatusFailed
	})

	if failed.ProgressState != ExportProgressStateFailed {
		t.Fatalf("expected failed progress state, got %q", failed.ProgressState)
	}
	if failed.ArtifactReadiness != ExportArtifactReadinessPartial {
		t.Fatalf("expected partial artifact readiness, got %q", failed.ArtifactReadiness)
	}
	if len(failed.Artifacts) != 1 {
		t.Fatalf("expected one ready artifact before failure, got %d", len(failed.Artifacts))
	}
	if failed.ProgressPct <= exportProgressMaterializeBasePct {
		t.Fatalf("expected failure after materialization progress, got %d", failed.ProgressPct)
	}
	if failed.ETASeconds != nil {
		t.Fatalf("expected nil eta for failed export, got %v", *failed.ETASeconds)
	}
}

func TestDeriveArtifactReadinessReadyWithoutDeclaredFormats(t *testing.T) {
	readiness := deriveArtifactReadiness(ExportRecord{
		Status:    ExportStatusFailed,
		Artifacts: []ExportArtifact{{ID: "artifact-1"}},
	})

	if readiness != ExportArtifactReadinessReady {
		t.Fatalf("expected ready artifact readiness without declared formats, got %q", readiness)
	}
}

func newBlockedExportRuntime() (*stubRuntime, <-chan struct{}, chan struct{}) {
	formatProvider := datasetapi.GetFormatProvider()
	runtime := newStubRuntime(
		"frog",
		"progress",
		"1",
		"Progress",
		"progress dataset",
		[]datasetapi.Column{{Name: "value", Type: "integer"}},
		[]datasetapi.Format{formatProvider.JSON()},
	)

	runStarted := make(chan struct{})
	releaseRun := make(chan struct{})
	runtime.runFn = func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
		close(runStarted)
		<-releaseRun
		return datasetapi.RunResult{
			Schema:      append([]datasetapi.Column(nil), runtime.desc.Columns...),
			Rows:        []datasetapi.Row{{"value": 11}},
			GeneratedAt: time.Unix(0, 0).UTC(),
			Format:      formatProvider.JSON(),
		}, nil, nil
	}

	return runtime, runStarted, releaseRun
}

func waitForExportRecord(t *testing.T, worker *Worker, id string, timeout time.Duration, predicate func(ExportRecord) bool) ExportRecord {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		record, ok := worker.GetExport(id)
		if ok && predicate(record) {
			return record
		}
		time.Sleep(10 * time.Millisecond)
	}

	record, ok := worker.GetExport(id)
	if !ok {
		t.Fatalf("export %s not found", id)
	}
	t.Fatalf("timed out waiting for export %s; last=%+v", id, record)
	return ExportRecord{}
}

type failAfterNStore struct {
	backing *MemoryObjectStore
	failAt  int

	mu    sync.Mutex
	calls int
}

func (s *failAfterNStore) Put(ctx context.Context, key string, payload []byte, contentType string, metadata map[string]any) (ExportArtifact, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()

	if call >= s.failAt {
		return ExportArtifact{}, fmt.Errorf("put failed on call %d", call)
	}
	return s.backing.Put(ctx, key, payload, contentType, metadata)
}

func (s *failAfterNStore) Get(ctx context.Context, key string) (ExportArtifact, []byte, error) {
	return s.backing.Get(ctx, key)
}

func (s *failAfterNStore) Delete(ctx context.Context, key string) (bool, error) {
	return s.backing.Delete(ctx, key)
}

func (s *failAfterNStore) List(ctx context.Context, prefix string) ([]ExportArtifact, error) {
	return s.backing.List(ctx, prefix)
}
