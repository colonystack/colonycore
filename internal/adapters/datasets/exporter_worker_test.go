package datasets

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

type stubRuntime struct {
	desc       datasetapi.TemplateDescriptor
	formats    map[datasetapi.Format]struct{}
	validateFn func(map[string]any) (map[string]any, []datasetapi.ParameterError)
	runFn      func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error)
}

func newStubRuntime(plugin, key, version, title, description string, columns []datasetapi.Column, formats []datasetapi.Format) *stubRuntime {
	desc := datasetapi.TemplateDescriptor{
		Plugin:        plugin,
		Key:           key,
		Version:       version,
		Title:         title,
		Description:   description,
		Columns:       append([]datasetapi.Column(nil), columns...),
		OutputFormats: append([]datasetapi.Format(nil), formats...),
		Slug:          fmt.Sprintf("%s/%s@%s", plugin, key, version),
	}
	formatSet := make(map[datasetapi.Format]struct{}, len(formats))
	for _, f := range formats {
		formatSet[f] = struct{}{}
	}
	return &stubRuntime{desc: desc, formats: formatSet}
}

func (s *stubRuntime) Descriptor() datasetapi.TemplateDescriptor {
	return s.desc
}

func (s *stubRuntime) SupportsFormat(format datasetapi.Format) bool {
	if s.formats == nil {
		return false
	}
	_, ok := s.formats[format]
	return ok
}

func (s *stubRuntime) ValidateParameters(params map[string]any) (map[string]any, []datasetapi.ParameterError) {
	if s.validateFn != nil {
		return s.validateFn(params)
	}
	return cloneParams(params), nil
}

func (s *stubRuntime) Run(ctx context.Context, params map[string]any, scope datasetapi.Scope, format datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
	if s.runFn != nil {
		return s.runFn(ctx, params, scope, format)
	}
	return datasetapi.RunResult{Format: format}, nil, nil
}

func cloneParams(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildRuntimeTemplate() *stubRuntime {
	runtime := newStubRuntime(
		"frog",
		"snap",
		"1",
		"Snap",
		"snap dataset",
		[]datasetapi.Column{{Name: "value", Type: "integer"}},
		[]datasetapi.Format{datasetapi.FormatJSON, datasetapi.FormatCSV, datasetapi.FormatHTML, datasetapi.FormatParquet, datasetapi.FormatPNG},
	)
	runtime.validateFn = func(params map[string]any) (map[string]any, []datasetapi.ParameterError) {
		if params == nil {
			return nil, nil
		}
		cleaned := cloneParams(params)
		if v, ok := params["limit"]; ok {
			switch val := v.(type) {
			case int, int64:
				cleaned["limit"] = val
			case float64:
				cleaned["limit"] = int(val)
			default:
				return nil, []datasetapi.ParameterError{{Name: "limit", Message: "must be integer"}}
			}
		}
		return cleaned, nil
	}
	runtime.runFn = func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
		return datasetapi.RunResult{
			Schema:      append([]datasetapi.Column(nil), runtime.desc.Columns...),
			Rows:        []datasetapi.Row{{"value": 42}},
			GeneratedAt: time.Unix(0, 0).UTC(),
			Format:      datasetapi.FormatJSON,
		}, nil, nil
	}
	return runtime
}

func buildFailRuntime() *stubRuntime {
	runtime := newStubRuntime(
		"frog",
		"fail",
		"1",
		"Fail",
		"fail dataset",
		[]datasetapi.Column{{Name: "value", Type: "integer"}},
		[]datasetapi.Format{datasetapi.FormatJSON},
	)
	runtime.runFn = func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
		return datasetapi.RunResult{}, nil, fmt.Errorf("boom run")
	}
	return runtime
}

func buildBadJSONRuntime() *stubRuntime {
	runtime := newStubRuntime(
		"frog",
		"badjson",
		"1",
		"Bad JSON",
		"bad json dataset",
		[]datasetapi.Column{{Name: "value", Type: "string"}},
		[]datasetapi.Format{datasetapi.FormatJSON},
	)
	runtime.runFn = func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
		return datasetapi.RunResult{
			Schema:      append([]datasetapi.Column(nil), runtime.desc.Columns...),
			Rows:        []datasetapi.Row{{"value": make(chan int)}},
			GeneratedAt: time.Unix(0, 0).UTC(),
			Format:      datasetapi.FormatJSON,
		}, nil, nil
	}
	return runtime
}

type fakeCatalog struct{ tpl datasetapi.TemplateRuntime }

func (f fakeCatalog) ResolveDatasetTemplate(slug string) (datasetapi.TemplateRuntime, bool) {
	if f.tpl != nil && f.tpl.Descriptor().Slug == slug {
		return f.tpl, true
	}
	return nil, false
}

func (f fakeCatalog) DatasetTemplates() []datasetapi.TemplateDescriptor {
	if f.tpl == nil {
		return nil
	}
	return []datasetapi.TemplateDescriptor{f.tpl.Descriptor()}
}

type transientCatalog struct {
	tpl    datasetapi.TemplateRuntime
	served bool
}

func (c *transientCatalog) ResolveDatasetTemplate(slug string) (datasetapi.TemplateRuntime, bool) {
	if !c.served && c.tpl != nil && slug == c.tpl.Descriptor().Slug {
		c.served = true
		return c.tpl, true
	}
	return nil, false
}

func (c *transientCatalog) DatasetTemplates() []datasetapi.TemplateDescriptor {
	if c.tpl == nil {
		return nil
	}
	return []datasetapi.TemplateDescriptor{c.tpl.Descriptor()}
}

type memAudit struct{ entries []AuditEntry }

func (m *memAudit) Record(_ context.Context, e AuditEntry) { m.entries = append(m.entries, e) }

func TestWorkerSuccessAcrossFormats(t *testing.T) {
	tpl := buildRuntimeTemplate()
	catalog := fakeCatalog{tpl: tpl}
	store := NewMemoryObjectStore()
	audit := &memAudit{}
	w := NewWorker(catalog, store, audit)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	rec, err := w.EnqueueExport(context.Background(), ExportInput{
		TemplateSlug: tpl.Descriptor().Slug,
		Formats: []datasetapi.Format{
			datasetapi.FormatJSON,
			datasetapi.FormatCSV,
			datasetapi.FormatHTML,
			datasetapi.FormatParquet,
			datasetapi.FormatPNG,
		},
		RequestedBy: "tester",
	})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := w.GetExport(rec.ID)
		if !ok {
			t.Fatalf("missing export")
		}
		if cur.Status == ExportStatusSucceeded {
			if len(cur.Artifacts) != 5 {
				t.Fatalf("expected 5 artifacts, got %d", len(cur.Artifacts))
			}
			return
		}
		if cur.Status == ExportStatusFailed {
			t.Fatalf("unexpected failure: %s", cur.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("export did not complete")
}

func TestWorkerParameterValidationFailure(t *testing.T) {
	tpl := buildRuntimeTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	rec, err := w.EnqueueExport(context.Background(), ExportInput{
		TemplateSlug: tpl.Descriptor().Slug,
		Formats:      []datasetapi.Format{datasetapi.FormatJSON},
		Parameters:   map[string]any{"limit": "not-int"},
	})
	if err != nil {
		t.Fatalf("enqueue unexpected error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := w.GetExport(rec.ID)
		if !ok {
			t.Fatalf("missing export record")
		}
		if cur.Status == ExportStatusFailed {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected failed status due to parameter validation")
}

func TestWorkerUnsupportedFormat(t *testing.T) {
	tpl := buildRuntimeTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	_, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{"bogus"}})
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestWorkerTemplateNotFound(t *testing.T) {
	catalog := fakeCatalog{tpl: buildRuntimeTemplate()}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	_, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: "missing/slug@1", Formats: []datasetapi.Format{datasetapi.FormatJSON}})
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestMaterializeUnsupportedFormat(t *testing.T) {
	tpl := buildRuntimeTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)

	_, err := w.materialize(datasetapi.Format("weird"), tpl, datasetapi.RunResult{Rows: []datasetapi.Row{{"value": 1}}})
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestWorkerRunFailure(t *testing.T) {
	tpl := buildFailRuntime()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{datasetapi.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := w.GetExport(rec.ID)
		if !ok {
			t.Fatalf("missing export record")
		}
		if cur.Status == ExportStatusFailed {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected failed status due to run error")
}

func TestWorkerQueueFull(t *testing.T) {
	tpl := buildRuntimeTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.queue = make(chan exportTask, 1)
	w.queue <- exportTask{id: "pre", input: ExportInput{TemplateSlug: tpl.Descriptor().Slug}}

	if _, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{datasetapi.FormatJSON}}); err == nil || !strings.Contains(err.Error(), "queue full") {
		t.Fatalf("expected queue full error, got %v", err)
	}
}

func TestWorkerProcessTemplateMissingSecondPass(t *testing.T) {
	tpl := buildRuntimeTemplate()
	cat := &transientCatalog{tpl: tpl}
	w := NewWorker(cat, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{datasetapi.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := w.GetExport(rec.ID)
		if !ok {
			t.Fatalf("missing export")
		}
		if cur.Status == ExportStatusFailed {
			if !strings.Contains(cur.Error, "template") {
				t.Fatalf("expected template missing error, got %s", cur.Error)
			}
			return
		}
		if cur.Status == ExportStatusSucceeded {
			t.Fatalf("expected failure due to template missing on process")
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("did not observe failure state")
}

func TestWorkerStoreArtifactFailure(t *testing.T) {
	tpl := buildRuntimeTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, errorStore{}, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{datasetapi.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := w.GetExport(rec.ID)
		if !ok {
			t.Fatalf("missing export record")
		}
		if cur.Status == ExportStatusFailed {
			if !strings.Contains(cur.Error, "store artifact failed") {
				t.Fatalf("unexpected error: %s", cur.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected failure due to store artifact error")
}

func TestWorkerProcessMissingRecordBranch(_ *testing.T) {
	tpl := buildRuntimeTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	w.queue <- exportTask{id: "ghost", input: ExportInput{TemplateSlug: tpl.Descriptor().Slug}}
	time.Sleep(50 * time.Millisecond)
}

func TestWorkerMaterializeJSONMarshalError(t *testing.T) {
	tpl := buildBadJSONRuntime()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()

	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []datasetapi.Format{datasetapi.FormatJSON}})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cur, ok := w.GetExport(rec.ID)
		if !ok {
			t.Fatalf("missing export")
		}
		if cur.Status == ExportStatusFailed {
			if !strings.Contains(cur.Error, "marshal json") {
				t.Fatalf("expected marshal json error, got %s", cur.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected JSON marshal failure not observed")
}

type errorStore struct{}

func (errorStore) Put(context.Context, string, []byte, string, map[string]any) (ExportArtifact, error) {
	return ExportArtifact{}, fmt.Errorf("put failed")
}

func (errorStore) Get(context.Context, string) (ExportArtifact, []byte, error) {
	return ExportArtifact{}, nil, fmt.Errorf("no")
}

func (errorStore) Delete(context.Context, string) (bool, error) { return false, nil }

func (errorStore) List(context.Context, string) ([]ExportArtifact, error) { return nil, nil }
