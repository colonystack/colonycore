package datasets

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"colonycore/internal/core"
)

// fake template implementing core.DatasetTemplate
func buildTemplate() core.DatasetTemplate {
	t := core.DatasetTemplate{
		Plugin: "frog", Key: "snap", Version: "1", Title: "Snap", Dialect: core.DatasetDialectSQL, Query: "SELECT 1",
		Parameters:    []core.DatasetParameter{{Name: "limit", Type: "integer", Required: false}},
		Columns:       []core.DatasetColumn{{Name: "value", Type: "integer"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatHTML, core.FormatParquet, core.FormatPNG},
	}
	core.BindTemplateForTests(&t, func(_ context.Context, _ core.DatasetRunRequest) (core.DatasetRunResult, error) {
		return core.DatasetRunResult{Rows: []map[string]any{{"value": 42}}, Schema: []core.DatasetColumn{{Name: "value", Type: "integer"}}, GeneratedAt: time.Unix(0, 0).UTC()}, nil
	})
	return t
}

type fakeCatalog struct{ tpl core.DatasetTemplate }

func (f fakeCatalog) ResolveDatasetTemplate(slug string) (core.DatasetTemplate, bool) {
	if slug == f.tpl.Descriptor().Slug {
		return f.tpl, true
	}
	return core.DatasetTemplate{}, false
}

func (f fakeCatalog) DatasetTemplates() []core.DatasetTemplateDescriptor {
	return []core.DatasetTemplateDescriptor{f.tpl.Descriptor()}
}

// simple audit collector
type memAudit struct{ entries []AuditEntry }

func (m *memAudit) Record(_ context.Context, e AuditEntry) { m.entries = append(m.entries, e) }

func TestWorkerSuccessAcrossFormats(t *testing.T) {
	tpl := buildTemplate()
	catalog := fakeCatalog{tpl: tpl}
	store := NewMemoryObjectStore()
	audit := &memAudit{}
	w := NewWorker(catalog, store, audit)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	// request all formats
	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON, core.FormatCSV, core.FormatHTML, core.FormatParquet, core.FormatPNG}, RequestedBy: "tester"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// spin until completed or timeout
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
	tpl := buildTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON}, Parameters: map[string]any{"limit": "not-int"}})
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
	tpl := buildTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	_, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{"bogus"}})
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

func TestWorkerTemplateNotFound(t *testing.T) {
	catalog := fakeCatalog{tpl: buildTemplate()}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	_, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: "missing/slug@1", Formats: []core.DatasetFormat{core.FormatJSON}})
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestMaterializeUnsupportedFormat(t *testing.T) {
	tpl := buildTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	// directly call materialize for unsupported format
	_, err := w.materialize("weird", tpl, core.DatasetRunResult{Rows: []map[string]any{{"value": 1}}})
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
}

// buildFailTemplate constructs a template whose binder returns an error when run.
func buildFailTemplate() core.DatasetTemplate {
	t := core.DatasetTemplate{
		Plugin: "frog", Key: "fail", Version: "1", Title: "Fail", Dialect: core.DatasetDialectSQL, Query: "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "integer"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
	}
	core.BindTemplateForTests(&t, func(_ context.Context, _ core.DatasetRunRequest) (core.DatasetRunResult, error) {
		return core.DatasetRunResult{}, fmt.Errorf("boom run")
	})
	return t
}

func TestWorkerRunFailure(t *testing.T) {
	tpl := buildFailTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON}})
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

// TestWorkerQueueFull covers the non-blocking enqueue default branch when queue channel is full.
func TestWorkerQueueFull(t *testing.T) {
	tpl := buildTemplate()
	catalog := fakeCatalog{tpl: tpl}
	w := NewWorker(catalog, nil, nil)
	// Replace queue with size 1 to force full condition quickly
	w.queue = make(chan exportTask, 1)
	// Pre-fill queue without starting worker so it remains full
	w.queue <- exportTask{id: "pre", input: ExportInput{TemplateSlug: tpl.Descriptor().Slug}}
	// Attempt enqueue should hit default path returning queue full error
	if _, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON}}); err == nil || !strings.Contains(err.Error(), "queue full") {
		t.Fatalf("expected queue full error, got %v", err)
	}
}

// errorStore implements ObjectStore that fails on Put.
type errorStore struct{}

func (errorStore) Put(context.Context, string, []byte, string, map[string]any) (ExportArtifact, error) {
	return ExportArtifact{}, fmt.Errorf("put failed")
}
func (errorStore) Get(context.Context, string) (ExportArtifact, []byte, error) {
	return ExportArtifact{}, nil, fmt.Errorf("no")
}
func (errorStore) Delete(context.Context, string) (bool, error)           { return false, nil }
func (errorStore) List(context.Context, string) ([]ExportArtifact, error) { return nil, nil }

// transientCatalog returns template first time then reports missing to exercise process missing template branch.
type transientCatalog struct {
	tpl    core.DatasetTemplate
	served bool
}

func (c *transientCatalog) ResolveDatasetTemplate(slug string) (core.DatasetTemplate, bool) {
	if !c.served && slug == c.tpl.Descriptor().Slug {
		c.served = true
		return c.tpl, true
	}
	return core.DatasetTemplate{}, false
}
func (c *transientCatalog) DatasetTemplates() []core.DatasetTemplateDescriptor {
	return []core.DatasetTemplateDescriptor{c.tpl.Descriptor()}
}

func TestWorkerProcessTemplateMissingSecondPass(t *testing.T) {
	tpl := buildTemplate()
	cat := &transientCatalog{tpl: tpl}
	w := NewWorker(cat, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON}})
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
	tpl := buildTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, errorStore{}, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON}})
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
	tpl := buildTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	// inject task with id not present in jobs map to hit record==nil early return
	w.queue <- exportTask{id: "ghost", input: ExportInput{TemplateSlug: tpl.Descriptor().Slug}}
	// allow loop iteration
	time.Sleep(50 * time.Millisecond)
	// nothing to assert; absence of panic and normal exit covers branch
}

// buildBadJSONTemplate returns a template whose binder yields a row containing an unsupported JSON type (channel) to force marshal failure.
func buildBadJSONTemplate() core.DatasetTemplate {
	t := core.DatasetTemplate{
		Plugin: "frog", Key: "badjson", Version: "1", Title: "Bad JSON", Dialect: core.DatasetDialectSQL, Query: "SELECT 1",
		Columns:       []core.DatasetColumn{{Name: "value", Type: "string"}},
		OutputFormats: []core.DatasetFormat{core.FormatJSON},
	}
	ch := make(chan int)
	core.BindTemplateForTests(&t, func(_ context.Context, _ core.DatasetRunRequest) (core.DatasetRunResult, error) {
		return core.DatasetRunResult{Rows: []map[string]any{{"value": ch}}, Schema: t.Columns, GeneratedAt: time.Unix(0, 0).UTC()}, nil
	})
	return t
}

func TestWorkerMaterializeJSONMarshalError(t *testing.T) {
	tpl := buildBadJSONTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	w.Start()
	defer func() { _ = w.Stop(context.Background()) }()
	rec, err := w.EnqueueExport(context.Background(), ExportInput{TemplateSlug: tpl.Descriptor().Slug, Formats: []core.DatasetFormat{core.FormatJSON}})
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
