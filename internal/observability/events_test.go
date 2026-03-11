package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
)

func TestNoopRecorder(_ *testing.T) {
	var recorder NoopRecorder
	recorder.Record(context.Background(), Event{Name: "ignored"})
}

func TestNewJSONRecorderNilWriter(t *testing.T) {
	recorder := NewJSONRecorder(nil, "test")
	if _, ok := recorder.(NoopRecorder); !ok {
		t.Fatalf("expected noop recorder for nil writer, got %T", recorder)
	}
}

func TestJSONRecorderDefaultsAndClones(t *testing.T) {
	var out bytes.Buffer
	recorder := NewJSONRecorder(&out, "unit")

	labels := map[string]string{"template_id": "frog/summary@1.0.0"}
	measures := map[string]float64{"rows": 2}
	recorder.Record(context.Background(), Event{
		Category: "catalog.operation",
		Name:     "template.run",
		Status:   StatusSuccess,
		Labels:   labels,
		Measures: measures,
	})

	labels["template_id"] = "mutated"
	measures["rows"] = 99

	var emitted Event
	if err := json.Unmarshal(out.Bytes(), &emitted); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if emitted.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("expected schema version %q, got %q", SchemaVersionV1, emitted.SchemaVersion)
	}
	if emitted.Source != "unit" {
		t.Fatalf("expected source unit, got %q", emitted.Source)
	}
	if emitted.Labels["template_id"] != "frog/summary@1.0.0" {
		t.Fatalf("expected cloned labels, got %+v", emitted.Labels)
	}
	if emitted.Measures["rows"] != 2 {
		t.Fatalf("expected cloned measures, got %+v", emitted.Measures)
	}
	if emitted.Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be set")
	}
}

func TestJSONRecorderPreservesExplicitValues(t *testing.T) {
	var out bytes.Buffer
	recorder := NewJSONRecorder(&out, "default-source")

	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC+2", 2*60*60))
	recorder.Record(context.Background(), Event{
		SchemaVersion: "custom.v1",
		Timestamp:     ts,
		Source:        "custom-source",
		Category:      CategoryRuleExecution,
		Name:          "rule.evaluate",
		Status:        StatusError,
		Error:         "boom",
	})

	var emitted Event
	if err := json.Unmarshal(out.Bytes(), &emitted); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if emitted.SchemaVersion != "custom.v1" {
		t.Fatalf("expected custom schema version, got %q", emitted.SchemaVersion)
	}
	if emitted.Source != "custom-source" {
		t.Fatalf("expected custom source, got %q", emitted.Source)
	}
	if !emitted.Timestamp.Equal(ts.UTC()) {
		t.Fatalf("expected UTC timestamp %v, got %v", ts.UTC(), emitted.Timestamp)
	}
}

func TestDurationMS(t *testing.T) {
	duration := 1500 * time.Microsecond
	if got := DurationMS(duration); got != 1.5 {
		t.Fatalf("expected 1.5ms, got %.3f", got)
	}
}

func TestJSONRecorderCallsOnError(t *testing.T) {
	expected := errors.New("encode failed")
	errs := make([]error, 0, 1)
	recorder := NewJSONRecorderWithOnError(errWriter{err: expected}, "unit", func(err error) {
		errs = append(errs, err)
	})

	recorder.Record(context.Background(), Event{
		Category: CategoryCatalogOperation,
		Name:     "catalog.add",
		Status:   StatusSuccess,
	})

	if len(errs) != 1 {
		t.Fatalf("expected one callback invocation, got %d", len(errs))
	}
	if !errors.Is(errs[0], expected) {
		t.Fatalf("expected callback error %v, got %v", expected, errs[0])
	}
}

type errWriter struct {
	err error
}

func (w errWriter) Write([]byte) (int, error) {
	if w.err == nil {
		return 0, io.ErrClosedPipe
	}
	return 0, w.err
}
