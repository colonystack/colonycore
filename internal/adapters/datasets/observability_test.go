package datasets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/internal/observability"
	"colonycore/pkg/datasetapi"
)

type captureDatasetEvents struct {
	events chan observability.Event
}

func newCaptureDatasetEvents() *captureDatasetEvents {
	return &captureDatasetEvents{
		events: make(chan observability.Event, 512),
	}
}

func (c *captureDatasetEvents) Record(_ context.Context, event observability.Event) {
	if c == nil || c.events == nil {
		return
	}
	select {
	case c.events <- event:
	default:
	}
}

func (c *captureDatasetEvents) await(name, status string, timeout time.Duration) bool {
	if c == nil {
		return false
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case event := <-c.events:
			if event.Name == name && event.Status == status {
				return true
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	return false
}

func TestHandlerRunEmitsEvents(t *testing.T) {
	tpl := buildTemplate()
	h := NewHandler(testCatalog{tpl: tpl})
	events := newCaptureDatasetEvents()
	h.Events = events

	d := tpl.Descriptor()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/run", strings.NewReader(`{"parameters":{}}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !events.await("catalog.template.run", observability.StatusSuccess, time.Second) {
		t.Fatalf("expected successful template run event")
	}
}

func TestHandlerRunEmitsErrorEvent(t *testing.T) {
	tpl := buildTemplate()
	tpl.OutputFormats = []datasetapi.Format{core.FormatJSON}
	core.BindTemplateForTests(&tpl, func(_ context.Context, _ datasetapi.RunRequest) (datasetapi.RunResult, error) {
		return datasetapi.RunResult{
			Schema:      tpl.Columns,
			Rows:        []datasetapi.Row{{"value": "ok"}},
			GeneratedAt: time.Unix(0, 0).UTC(),
			Format:      core.FormatJSON,
		}, nil
	})
	h := NewHandler(testCatalog{tpl: tpl})
	events := newCaptureDatasetEvents()
	h.Events = events

	d := tpl.Descriptor()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/"+d.Plugin+"/"+d.Key+"/"+d.Version+"/run?format=csv", strings.NewReader(`{"parameters":{}}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotAcceptable {
		t.Fatalf("expected 406, got %d", rec.Code)
	}
	if !events.await("catalog.template.run", observability.StatusError, time.Second) {
		t.Fatalf("expected template run error event")
	}
}

func TestWorkerEnqueueEmitsEvents(t *testing.T) {
	tpl := buildRuntimeTemplate()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	events := newCaptureDatasetEvents()
	w.events = events

	_, err := w.EnqueueExport(context.Background(), ExportInput{
		TemplateSlug: tpl.Descriptor().Slug,
		Formats:      []datasetapi.Format{datasetapi.GetFormatProvider().JSON()},
	})
	if err != nil {
		t.Fatalf("expected enqueue success, got %v", err)
	}
	if !events.await("catalog.export.enqueue", observability.StatusQueued, time.Second) {
		t.Fatalf("expected enqueue queued event")
	}

	_, err = w.EnqueueExport(context.Background(), ExportInput{
		TemplateSlug: tpl.Descriptor().Slug,
		Formats:      []datasetapi.Format{datasetapi.Format("bogus")},
	})
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
	if !events.await("catalog.export.enqueue", observability.StatusError, time.Second) {
		t.Fatalf("expected enqueue error event")
	}
}

func TestWorkerFailureEmitsProcessErrorEvent(t *testing.T) {
	tpl := buildFailRuntime()
	w := NewWorker(fakeCatalog{tpl: tpl}, nil, nil)
	events := newCaptureDatasetEvents()
	w.events = events
	w.Start()
	t.Cleanup(func() { _ = w.Stop(context.Background()) })

	record, err := w.EnqueueExport(context.Background(), ExportInput{
		TemplateSlug: tpl.Descriptor().Slug,
		Formats:      []datasetapi.Format{datasetapi.GetFormatProvider().JSON()},
	})
	if err != nil {
		t.Fatalf("enqueue export: %v", err)
	}

	waitForExportStatus(t, w, record.ID, ExportStatusFailed, 5*time.Second)

	if !events.await("catalog.export.process", observability.StatusError, 2*time.Second) {
		t.Fatalf("expected process error event")
	}
}

func waitForExportStatus(t *testing.T, worker *Worker, id string, expected ExportStatus, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	sleep := 25 * time.Millisecond
	const maxSleep = 250 * time.Millisecond
	for time.Now().Before(deadline) {
		current, ok := worker.GetExport(id)
		if !ok {
			t.Fatalf("missing export record %s", id)
		}
		if current.Status == expected {
			return
		}
		time.Sleep(sleep)
		if sleep < maxSleep {
			sleep *= 2
			if sleep > maxSleep {
				sleep = maxSleep
			}
		}
	}
	t.Fatalf("timed out waiting for export %s to reach status %s", id, expected)
}
