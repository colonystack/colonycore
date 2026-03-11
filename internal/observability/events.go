// Package observability defines structured event primitives and recorders used
// by ColonyCore runtime and CLI instrumentation.
package observability

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"
)

const (
	// SchemaVersionV1 identifies the first stable event schema revision.
	SchemaVersionV1 = "colonycore.observability.v1"

	// CategoryRegistryValidation groups registry-check validation events.
	CategoryRegistryValidation = "registry.validation"
	// CategoryPluginLifecycle groups plugin loading and registration events.
	CategoryPluginLifecycle = "plugin.lifecycle"
	// CategoryRuleExecution groups rule invocation and result events.
	CategoryRuleExecution = "rule.execution"
	// CategoryCatalogOperation groups catalog/template/export events.
	CategoryCatalogOperation = "catalog.operation"

	// StatusStart marks the beginning of an operation.
	StatusStart = "start"
	// StatusQueued marks work that is accepted but not yet running.
	StatusQueued = "queued"
	// StatusRunning marks work that is actively executing.
	StatusRunning = "running"
	// StatusSuccess marks a successful operation.
	StatusSuccess = "success"
	// StatusError marks a failed operation.
	StatusError = "error"
)

// Event is the canonical structured observability record emitted by ColonyCore.
type Event struct {
	SchemaVersion string             `json:"schema_version"`
	Timestamp     time.Time          `json:"timestamp"`
	Source        string             `json:"source"`
	Category      string             `json:"category"`
	Name          string             `json:"name"`
	Status        string             `json:"status"`
	DurationMS    float64            `json:"duration_ms,omitempty"`
	Error         string             `json:"error,omitempty"`
	Labels        map[string]string  `json:"labels,omitempty"`
	Measures      map[string]float64 `json:"measures,omitempty"`
}

// Recorder accepts structured events.
type Recorder interface {
	Record(ctx context.Context, event Event)
}

// NoopRecorder discards all events.
type NoopRecorder struct{}

// Record drops events.
func (NoopRecorder) Record(context.Context, Event) {}

// JSONRecorder encodes events as JSON lines to an io.Writer.
type JSONRecorder struct {
	mu      sync.Mutex
	enc     *json.Encoder
	source  string
	onError func(error)
}

// NewJSONRecorder builds a recorder that writes one JSON object per line.
// If writer is nil, a NoopRecorder is returned.
func NewJSONRecorder(writer io.Writer, source string) Recorder {
	return NewJSONRecorderWithOnError(writer, source, nil)
}

// NewJSONRecorderWithOnError builds a JSON recorder with an optional encode
// error callback.
func NewJSONRecorderWithOnError(writer io.Writer, source string, onError func(error)) Recorder {
	if writer == nil {
		return NoopRecorder{}
	}
	if onError == nil {
		onError = func(error) {}
	}
	return &JSONRecorder{
		enc:     json.NewEncoder(writer),
		source:  source,
		onError: onError,
	}
}

// Record emits one event encoded as JSON.
func (r *JSONRecorder) Record(_ context.Context, event Event) {
	if r == nil {
		return
	}
	normalized := normalizeEvent(event, r.source)
	r.mu.Lock()
	err := r.enc.Encode(normalized)
	r.mu.Unlock()
	if err != nil {
		r.onError(err)
	}
}

// DurationMS converts a time.Duration to milliseconds as a float64.
// Durations larger than about 1.8e+11ms (~104 days) lose integer precision in
// float64; this is acceptable for typical observability latency windows.
// Callers that need exact millisecond values should keep integer millisecond
// counters or use time.Duration arithmetic directly.
func DurationMS(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func normalizeEvent(event Event, defaultSource string) Event {
	if event.SchemaVersion == "" {
		event.SchemaVersion = SchemaVersionV1
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	} else {
		event.Timestamp = event.Timestamp.UTC()
	}
	if event.Source == "" {
		event.Source = defaultSource
	}
	event.Labels = cloneLabels(event.Labels)
	event.Measures = cloneMeasures(event.Measures)
	return event
}

func cloneLabels(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneMeasures(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]float64, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
