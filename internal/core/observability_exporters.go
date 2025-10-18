package core

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

var expvarSeq uint64

// ExpvarMetricsRecorder publishes aggregate timing and result counters via expvar.
// It fulfills MetricsRecorder for deployments that prefer process-local metrics
// without external dependencies. The recorder maintains totals in milliseconds
// per operation and success/error counters.
type ExpvarMetricsRecorder struct {
	name      string
	mu        sync.Mutex
	durations map[string]float64
	results   map[string]map[string]int64
}

// ExpvarMetricsSnapshot captures a read-only view of the recorded metrics.
type ExpvarMetricsSnapshot struct {
	DurationsMS map[string]float64          `json:"durations_ms_total"`
	Results     map[string]map[string]int64 `json:"results_total"`
	RecordedAt  time.Time                   `json:"recorded_at"`
}

// NewExpvarMetricsRecorder constructs an expvar-backed recorder and publishes it
// under the supplied name. When name is empty, a unique identifier is generated.
func NewExpvarMetricsRecorder(name string) *ExpvarMetricsRecorder {
	if name == "" {
		id := atomic.AddUint64(&expvarSeq, 1)
		name = fmt.Sprintf("core_service_metrics_%d", id)
	}
	rec := &ExpvarMetricsRecorder{
		name:      name,
		durations: make(map[string]float64),
		results:   make(map[string]map[string]int64),
	}
	expvar.Publish(name, expvar.Func(func() any {
		return rec.Snapshot()
	}))
	return rec
}

// Name returns the expvar export name associated with the recorder.
func (r *ExpvarMetricsRecorder) Name() string {
	return r.name
}

// Snapshot returns an immutable copy of the aggregated metrics.
func (r *ExpvarMetricsRecorder) Snapshot() ExpvarMetricsSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	durations := make(map[string]float64, len(r.durations))
	for op, total := range r.durations {
		durations[op] = total
	}

	results := make(map[string]map[string]int64, len(r.results))
	for op, statusCounts := range r.results {
		cpy := make(map[string]int64, len(statusCounts))
		for status, count := range statusCounts {
			cpy[status] = count
		}
		results[op] = cpy
	}

	return ExpvarMetricsSnapshot{
		DurationsMS: durations,
		Results:     results,
		RecordedAt:  time.Now().UTC(),
	}
}

// Observe records a service operation outcome.
func (r *ExpvarMetricsRecorder) Observe(_ context.Context, operation string, success bool, duration time.Duration) {
	if operation == "" {
		return
	}
	ms := float64(duration) / float64(time.Millisecond)
	status := "error"
	if success {
		status = "success"
	}

	r.mu.Lock()
	r.durations[operation] += ms
	if _, ok := r.results[operation]; !ok {
		r.results[operation] = make(map[string]int64, 2)
	}
	r.results[operation][status]++
	r.mu.Unlock()
}

// JSONTraceEntry represents a serialized trace span emitted by JSONTraceTracer.
type JSONTraceEntry struct {
	Operation  string    `json:"operation"`
	Status     string    `json:"status"`
	DurationMS float64   `json:"duration_ms"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

// JSONTraceTracer serializes spans to a writer and retains them for inspection.
type JSONTraceTracer struct {
	mu      sync.Mutex
	entries []JSONTraceEntry
	enc     *json.Encoder
}

// NewJSONTracer constructs a tracer that writes spans as JSON lines to the writer.
// The tracer retains all encoded spans for later inspection via Entries().
func NewJSONTracer(w io.Writer) *JSONTraceTracer {
	var enc *json.Encoder
	if w != nil {
		enc = json.NewEncoder(w)
	}
	return &JSONTraceTracer{
		enc: enc,
	}
}

// Entries returns a copy of all recorded spans.
func (t *JSONTraceTracer) Entries() []JSONTraceEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]JSONTraceEntry, len(t.entries))
	copy(out, t.entries)
	return out
}

// Start implements the Tracer interface.
func (t *JSONTraceTracer) Start(ctx context.Context, operation string) (context.Context, TraceSpan) {
	span := &jsonTraceSpan{
		tracer:    t,
		operation: operation,
		started:   time.Now().UTC(),
	}
	return ctx, span
}

type jsonTraceSpan struct {
	tracer    *JSONTraceTracer
	operation string
	started   time.Time
}

func (s *jsonTraceSpan) End(err error) {
	status := "success"
	var errMsg string
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}
	ended := time.Now().UTC()
	entry := JSONTraceEntry{
		Operation:  s.operation,
		Status:     status,
		DurationMS: float64(ended.Sub(s.started)) / float64(time.Millisecond),
		Error:      errMsg,
		StartedAt:  s.started,
		EndedAt:    ended,
	}

	s.tracer.mu.Lock()
	s.tracer.entries = append(s.tracer.entries, entry)
	if s.tracer.enc != nil {
		_ = s.tracer.enc.Encode(entry)
	}
	s.tracer.mu.Unlock()
}
