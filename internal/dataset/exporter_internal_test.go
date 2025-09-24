package dataset

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"colonycore/internal/core"
)

type stringer struct{}

func (stringer) String() string { return "stringer" }

func TestMaterializeFormats(t *testing.T) {
	worker := NewWorker(nil, nil, nil)
	template := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "population",
		Version:       "1.0.0",
		Title:         "Population",
		Description:   "demo",
		OutputFormats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatHTML, core.FormatParquet, core.FormatPNG},
		Columns: []core.DatasetColumn{{
			Name: "value",
			Type: "string",
		}},
	}
	result := core.DatasetRunResult{
		Schema: []core.DatasetColumn{{Name: "value", Type: "string"}},
		Rows: []map[string]any{{
			"value": "alpha",
		}},
		GeneratedAt: time.Now().UTC(),
		Format:      core.FormatJSON,
	}

	formats := []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatHTML, core.FormatParquet, core.FormatPNG}
	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			rendered, err := worker.materialize(format, template, result)
			if err != nil {
				t.Fatalf("materialize %s: %v", format, err)
			}
			if len(rendered.Payload) == 0 {
				t.Fatalf("expected payload for %s", format)
			}
			if rendered.Artifact.Format != format {
				t.Fatalf("unexpected artifact format: %s", rendered.Artifact.Format)
			}
		})
	}

	if _, err := worker.materialize(core.DatasetFormat("geojson"), template, result); err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestBuildPNGEmpty(t *testing.T) {
	result := core.DatasetRunResult{}
	pngPayload, err := buildPNG(result)
	if err != nil {
		t.Fatalf("buildPNG: %v", err)
	}
	if len(pngPayload) == 0 {
		t.Fatalf("expected png payload")
	}
}

func TestWorkerFailUpdatesRecord(t *testing.T) {
	worker := NewWorker(nil, nil, &MemoryAuditLog{})
	id := "export"
	worker.mu.Lock()
	worker.jobs[id] = &ExportRecord{
		ID:          id,
		RequestedBy: "analyst",
		Template: core.DatasetTemplateDescriptor{
			Slug: "frog/population@1.0.0",
		},
	}
	worker.mu.Unlock()

	worker.fail(id, "validation failed")

	record, ok := worker.GetExport(id)
	if !ok {
		t.Fatalf("expected export record")
	}
	if record.Status != ExportStatusFailed {
		t.Fatalf("expected failed status, got %s", record.Status)
	}
	if record.Error == "" {
		t.Fatalf("expected error message to be recorded")
	}
}

func TestFormatValueVariants(t *testing.T) {
	timeVal := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name   string
		in     any
		expect string
	}{
		{"nil", nil, ""},
		{"time", timeVal, timeVal.Format(time.RFC3339)},
		{"stringer", stringer{}, "stringer"},
		{"float32", float32(1.5), "1.5"},
		{"float64", float64(2.5), "2.5"},
		{"int", 7, "7"},
		{"int64", int64(9), "9"},
		{"bool", true, "true"},
		{"default", []int{1, 2}, "[1 2]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatValue(tc.in); got != tc.expect {
				t.Fatalf("expected %q, got %q", tc.expect, got)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusBadRequest, "bad request")

	if status := recorder.Result().StatusCode; status != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", status)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "bad request") {
		t.Fatalf("expected error body, got %s", body)
	}
}

type failingStore struct{}

func (f failingStore) Put(context.Context, string, []byte, string, map[string]any) (ExportArtifact, error) {
	return ExportArtifact{}, fmt.Errorf("store offline")
}

func (f failingStore) Get(context.Context, string) (ExportArtifact, []byte, error) {
	return ExportArtifact{}, nil, fmt.Errorf("store offline")
}

func (f failingStore) Delete(context.Context, string) (bool, error) { return false, nil }

func (f failingStore) List(context.Context, string) ([]ExportArtifact, error) { return nil, nil }

type testDatasetPlugin struct {
	dataset core.DatasetTemplate
}

func (p testDatasetPlugin) Name() string { return "test-dataset" }

func (p testDatasetPlugin) Version() string { return "0.0.1" }

func (p testDatasetPlugin) Register(registry *core.PluginRegistry) error {
	return registry.RegisterDatasetTemplate(p.dataset)
}

func TestWorkerProcessParameterFailure(t *testing.T) {
	template := core.DatasetTemplate{
		Plugin:      "frog",
		Key:         "fail",
		Version:     "1.0.0",
		Title:       "Fail",
		Description: "missing params",
		Dialect:     core.DatasetDialectSQL,
		Query:       "SELECT 1",
		Parameters: []core.DatasetParameter{{
			Name:     "required",
			Type:     "string",
			Required: true,
		}},
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	worker := NewWorker(svc, nil, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() {
		_ = worker.Stop(context.Background())
	})

	slug := svc.DatasetTemplates()[0].Slug
	queued, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug, RequestedBy: "analyst", Formats: []core.DatasetFormat{core.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := worker.GetExport(queued.ID)
		if record.Status == ExportStatusFailed {
			if !strings.Contains(record.Error, "required parameter") {
				t.Fatalf("expected parameter failure, got %s", record.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not fail in time: %+v", record)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWorkerProcessStoreFailure(t *testing.T) {
	template := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "store",
		Version:       "1.0.0",
		Title:         "Store",
		Description:   "store failure",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{
					Rows: []map[string]any{{"value": "ok"}},
				}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	store := failingStore{}
	worker := NewWorker(svc, store, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() {
		_ = worker.Stop(context.Background())
	})

	slug := svc.DatasetTemplates()[0].Slug
	queued, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug, RequestedBy: "analyst"})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := worker.GetExport(queued.ID)
		if record.Status == ExportStatusFailed {
			if !strings.Contains(record.Error, "store artifact failed") {
				t.Fatalf("expected store failure, got %s", record.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not fail due to store error in time: %+v", record)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWorkerProcessSuccessMultipleFormats(t *testing.T) {
	template := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "success",
		Version:       "1.0.0",
		Title:         "Success",
		Description:   "multi format",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatHTML},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{
					Rows: []map[string]any{{"value": "ok"}},
				}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	store := NewMemoryObjectStore()
	audit := &MemoryAuditLog{}
	worker := NewWorker(svc, store, audit)
	worker.Start()
	t.Cleanup(func() {
		_ = worker.Stop(context.Background())
	})

	slug := svc.DatasetTemplates()[0].Slug
	formats := []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatHTML}
	queued, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug, RequestedBy: "analyst", Formats: formats})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := worker.GetExport(queued.ID)
		if record.Status == ExportStatusSucceeded {
			if len(record.Artifacts) != len(formats) {
				t.Fatalf("expected %d artifacts, got %d", len(formats), len(record.Artifacts))
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not succeed in time: %+v", record)
		}
		time.Sleep(20 * time.Millisecond)
	}

	if len(store.Objects()) == 0 {
		t.Fatalf("expected artifacts stored")
	}
}

type flakyCatalog struct {
	template core.DatasetTemplate
	used     bool
}

func (f *flakyCatalog) DatasetTemplates() []core.DatasetTemplateDescriptor {
	return []core.DatasetTemplateDescriptor{f.template.Descriptor()}
}

func (f *flakyCatalog) ResolveDatasetTemplate(slug string) (core.DatasetTemplate, bool) {
	if !f.used && slug == f.template.Descriptor().Slug {
		f.used = true
		return f.template, true
	}
	return core.DatasetTemplate{}, false
}

func TestWorkerProcessTemplateMissing(t *testing.T) {
	template := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "missing",
		Version:       "1.0.0",
		Title:         "Missing",
		Description:   "catalog drops template",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	flaky := &flakyCatalog{template: template}
	worker := NewWorker(flaky, nil, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() {
		closeCh := make(chan struct{})
		go func() {
			_ = worker.Stop(context.Background())
			close(closeCh)
		}()
		select {
		case <-closeCh:
		case <-time.After(time.Second):
			t.Fatalf("worker did not stop")
		}
	})

	descriptor := flaky.DatasetTemplates()[0]
	queued, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: descriptor.Slug, RequestedBy: "analyst", Formats: []core.DatasetFormat{core.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := worker.GetExport(queued.ID)
		if record.Status == ExportStatusFailed {
			if !strings.Contains(record.Error, "template") {
				t.Fatalf("expected template missing error, got %s", record.Error)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not fail for missing template in time: %+v", record)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWorkerActorTemplateScopeFallback(t *testing.T) {
	worker := NewWorker(nil, nil, nil)
	if actor := worker.actorFor("unknown"); actor != "" {
		t.Fatalf("expected empty actor, got %s", actor)
	}
	if slug := worker.templateFor("unknown"); slug != "" {
		t.Fatalf("expected empty template slug, got %s", slug)
	}
	if scope := worker.scopeFor("unknown"); scope.Requestor != "" || len(scope.ProjectIDs) != 0 {
		t.Fatalf("expected zero-value scope, got %+v", scope)
	}
}

func TestWorkerProcessSnapshotMissing(t *testing.T) {
	worker := NewWorker(nil, nil, nil)
	worker.process(exportTask{id: "missing"})
}

type staticCatalog struct {
	template core.DatasetTemplate
}

func (s staticCatalog) DatasetTemplates() []core.DatasetTemplateDescriptor {
	return []core.DatasetTemplateDescriptor{s.template.Descriptor()}
}

func (s staticCatalog) ResolveDatasetTemplate(slug string) (core.DatasetTemplate, bool) {
	if slug == s.template.Descriptor().Slug {
		return s.template, true
	}
	return core.DatasetTemplate{}, false
}

func TestWorkerEnqueueQueueFull(t *testing.T) {
	templateDef := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "queue",
		Version:       "1.0.0",
		Title:         "Queue",
		Description:   "queue fill",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: templateDef}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := svc.DatasetTemplates()[0]
	bound, ok := svc.ResolveDatasetTemplate(descriptor.Slug)
	if !ok {
		t.Fatalf("resolve template")
	}
	catalog := staticCatalog{template: bound}
	worker := NewWorker(catalog, nil, nil)
	for i := 0; i < cap(worker.queue); i++ {
		if _, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: descriptor.Slug}); err != nil {
			t.Fatalf("unexpected enqueue error at %d: %v", i, err)
		}
	}
	if _, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: descriptor.Slug}); err == nil {
		t.Fatalf("expected queue full error")
	}
}

func TestWorkerProcessSuccessWithoutStore(t *testing.T) {
	templateDef := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "nostore",
		Version:       "1.0.0",
		Title:         "No Store",
		Description:   "no object store",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: templateDef}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := svc.DatasetTemplates()[0]
	bound, ok := svc.ResolveDatasetTemplate(descriptor.Slug)
	if !ok {
		t.Fatalf("resolve template")
	}
	catalog := staticCatalog{template: bound}
	worker := NewWorker(catalog, nil, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() { _ = worker.Stop(context.Background()) })
	queued, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: descriptor.Slug, RequestedBy: "analyst", Formats: []core.DatasetFormat{core.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := worker.GetExport(queued.ID)
		if record.Status == ExportStatusSucceeded {
			if len(record.Artifacts) != 1 {
				t.Fatalf("expected single artifact without store, got %d", len(record.Artifacts))
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not succeed without store in time: %+v", record)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

type zeroStore struct{}

func (zeroStore) Put(context.Context, string, []byte, string, map[string]any) (ExportArtifact, error) {
	return ExportArtifact{ID: newID()}, nil
}

func (zeroStore) Get(context.Context, string) (ExportArtifact, []byte, error) {
	return ExportArtifact{}, nil, fmt.Errorf("not found")
}

func (zeroStore) Delete(context.Context, string) (bool, error) { return false, nil }

func (zeroStore) List(context.Context, string) ([]ExportArtifact, error) { return nil, nil }

func TestWorkerProcessStoreNormalization(t *testing.T) {
	templateDef := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "normalize",
		Version:       "1.0.0",
		Title:         "Normalize",
		Description:   "normalize stored artifacts",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: templateDef}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := svc.DatasetTemplates()[0]
	bound, ok := svc.ResolveDatasetTemplate(descriptor.Slug)
	if !ok {
		t.Fatalf("resolve template")
	}
	catalog := staticCatalog{template: bound}
	worker := NewWorker(catalog, zeroStore{}, &MemoryAuditLog{})
	worker.Start()
	t.Cleanup(func() { _ = worker.Stop(context.Background()) })
	queued, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: descriptor.Slug, RequestedBy: "analyst", Formats: []core.DatasetFormat{core.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		record, _ := worker.GetExport(queued.ID)
		if record.Status == ExportStatusSucceeded {
			if len(record.Artifacts) != 1 {
				t.Fatalf("expected normalized artifact, got %d", len(record.Artifacts))
			}
			artifact := record.Artifacts[0]
			if artifact.ContentType != "application/json" || artifact.SizeBytes == 0 {
				t.Fatalf("expected content type and size to be set, got %+v", artifact)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("export did not complete in time: %+v", record)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWorkerEnqueueRequiresSlug(t *testing.T) {
	worker := NewWorker(staticCatalog{}, nil, nil)
	if _, err := worker.EnqueueExport(context.Background(), ExportInput{}); err == nil {
		t.Fatalf("expected error when slug missing")
	}
}

func TestWorkerEnqueueCatalogNil(t *testing.T) {
	worker := NewWorker(nil, nil, nil)
	if _, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: "frog/demo@1.0.0"}); err == nil {
		t.Fatalf("expected error when catalog missing")
	}
}

func TestWorkerEnqueueDeduplicatesFormats(t *testing.T) {
	templateDef := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "dedupe",
		Version:       "1.0.0",
		Title:         "Dedupe",
		Description:   "dedupe formats",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				return core.DatasetRunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: templateDef}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	descriptor := svc.DatasetTemplates()[0]
	bound, ok := svc.ResolveDatasetTemplate(descriptor.Slug)
	if !ok {
		t.Fatalf("resolve template")
	}
	catalog := staticCatalog{template: bound}
	worker := NewWorker(catalog, nil, nil)
	record, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: descriptor.Slug, Formats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}
	if len(record.Formats) != 2 {
		t.Fatalf("expected deduplicated formats, got %v", record.Formats)
	}
}

func TestWorkerStopContextDeadline(t *testing.T) {
	block := make(chan struct{})
	started := make(chan struct{})
	template := core.DatasetTemplate{
		Plugin:        "frog",
		Key:           "stop",
		Version:       "1.0.0",
		Title:         "Stop",
		Description:   "blocking runner",
		Dialect:       core.DatasetDialectSQL,
		Query:         "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
		Binder: func(core.DatasetEnvironment) (core.DatasetRunner, error) {
			return func(context.Context, core.DatasetRunRequest) (core.DatasetRunResult, error) {
				close(started)
				<-block
				return core.DatasetRunResult{Rows: []map[string]any{{"value": "ok"}}}, nil
			}, nil
		},
	}
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}
	worker := NewWorker(svc, NewMemoryObjectStore(), &MemoryAuditLog{})
	worker.Start()
	slug := svc.DatasetTemplates()[0].Slug
	if _, err := worker.EnqueueExport(context.Background(), ExportInput{TemplateSlug: slug, RequestedBy: "analyst", Formats: []core.DatasetFormat{core.FormatJSON}}); err != nil {
		t.Fatalf("enqueue export: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatalf("runner did not start in time")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := worker.Stop(ctx); err == nil {
		t.Fatalf("expected context deadline error from Stop")
	}

	close(block)
	if err := worker.Stop(context.Background()); err != nil {
		t.Fatalf("second stop should succeed: %v", err)
	}
}
