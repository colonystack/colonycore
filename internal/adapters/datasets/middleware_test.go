package datasets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"colonycore/pkg/datasetapi"
)

type capturedRequestLog struct {
	level  string
	msg    string
	fields map[string]interface{}
}

type captureRequestLogger struct {
	entries []capturedRequestLog
}

func (l *captureRequestLogger) Info(msg string, args ...interface{}) {
	l.record("info", msg, args...)
}

func (l *captureRequestLogger) Error(msg string, args ...interface{}) {
	l.record("error", msg, args...)
}

func (l *captureRequestLogger) record(level, msg string, args ...interface{}) {
	fields := make(map[string]interface{}, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		fields[key] = args[i+1]
	}
	l.entries = append(l.entries, capturedRequestLog{
		level:  level,
		msg:    msg,
		fields: fields,
	})
}

func TestHandlerErrorsUseProblemJSON(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, datasetTemplatesPath, nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("expected problem content type, got %q", got)
	}

	var problem problemDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Type != problemTypeBlank {
		t.Fatalf("expected problem type %q, got %q", problemTypeBlank, problem.Type)
	}
	if problem.Title != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("expected title %q, got %q", http.StatusText(http.StatusInternalServerError), problem.Title)
	}
	if problem.Status != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, problem.Status)
	}
	if problem.Detail != "dataset catalog not configured" {
		t.Fatalf("expected detail %q, got %q", "dataset catalog not configured", problem.Detail)
	}
}

func TestHandlerCorrelationIDPropagatesToResponseAndLogs(t *testing.T) {
	logger := &captureRequestLogger{}
	h := NewHandler(testCatalog{tpl: buildTemplate()})
	h.Logger = logger

	req := httptest.NewRequest(http.MethodGet, datasetTemplatesPath, nil)
	req.Header.Set(correlationIDHeader, "corr-123")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get(correlationIDHeader); got != "corr-123" {
		t.Fatalf("expected correlation id %q, got %q", "corr-123", got)
	}
	if len(logger.entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(logger.entries))
	}
	if got := logger.entries[0].fields["correlation_id"]; got != "corr-123" {
		t.Fatalf("expected log correlation id %q, got %#v", "corr-123", got)
	}
}

func TestHandlerGeneratesCorrelationIDWhenMissing(t *testing.T) {
	const generatedCorrelationID = "generated-123"

	logger := &captureRequestLogger{}
	h := NewHandler(testCatalog{tpl: buildTemplate()})
	h.Logger = logger
	h.CorrelationIDGenerator = func() string { return generatedCorrelationID }

	req := httptest.NewRequest(http.MethodGet, datasetTemplatesPath, nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if got := rec.Header().Get(correlationIDHeader); got != generatedCorrelationID {
		t.Fatalf("expected generated correlation id %q, got %q", generatedCorrelationID, got)
	}
	if len(logger.entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(logger.entries))
	}
	if got := logger.entries[0].fields["correlation_id"]; got != generatedCorrelationID {
		t.Fatalf("expected log correlation id %q, got %#v", generatedCorrelationID, got)
	}
}

func TestHandlerRecordsHTTPMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics, err := NewHTTPMetrics(registry)
	if err != nil {
		t.Fatalf("new metrics: %v", err)
	}

	h := NewHandler(testCatalog{tpl: buildTemplate()})
	h.Metrics = metrics

	req := httptest.NewRequest(http.MethodGet, datasetTemplatesPath, nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	if testutil.ToFloat64(metrics.requests.WithLabelValues(
		http.MethodGet,
		datasetTemplatesPath,
		"200",
	)) != 1 {
		t.Fatalf("expected request counter sample to equal 1")
	}

	foundDuration := false
	for _, family := range families {
		if family.GetName() != "http_request_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			var matched int
			for _, label := range metric.GetLabel() {
				switch {
				case label.GetName() == "method" && label.GetValue() == http.MethodGet:
					matched++
				case label.GetName() == "route" && label.GetValue() == datasetTemplatesPath:
					matched++
				case label.GetName() == "status_code" && label.GetValue() == "200":
					matched++
				}
			}
			if matched == 3 && metric.GetHistogram().GetSampleCount() == 1 {
				foundDuration = true
				break
			}
		}
	}
	if !foundDuration {
		var names []string
		for _, family := range families {
			names = append(names, family.GetName())
		}
		t.Fatalf("expected duration histogram sample count to equal 1; gathered metrics: %s", strings.Join(names, ", "))
	}
}

func TestNoopRequestLoggerAndCorrelationHelpers(t *testing.T) {
	logger := noopRequestLogger{}
	logger.Info("info", "key", "value")
	logger.Error("error", "key", "value")

	if got := CorrelationIDFromContext(nil); got != "" {
		t.Fatalf("expected empty correlation id for nil context, got %q", got)
	}
	if got := CorrelationIDFromContext(withCorrelationID(context.Background(), "")); got != "" {
		t.Fatalf("expected empty correlation id for blank value, got %q", got)
	}

	ctx := withCorrelationID(context.Background(), "corr-helper")
	if got := CorrelationIDFromContext(ctx); got != "corr-helper" {
		t.Fatalf("expected correlation id %q, got %q", "corr-helper", got)
	}
}

func TestStatusCapturingResponseWriterTracksStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	writer := &statusCapturingResponseWriter{ResponseWriter: rec}

	if got := writer.StatusCode(); got != http.StatusOK {
		t.Fatalf("expected default status 200, got %d", got)
	}

	writer.WriteHeader(http.StatusCreated)
	writer.WriteHeader(http.StatusAccepted)
	if got := writer.StatusCode(); got != http.StatusCreated {
		t.Fatalf("expected first status to win, got %d", got)
	}
}

func TestProblemAndValidationHelpersCoverFallbacks(t *testing.T) {
	const expectedErrorTitle = "Error"

	rec := httptest.NewRecorder()
	writeProblem(rec, 599, "")

	if rec.Code != 599 {
		t.Fatalf("expected 599, got %d", rec.Code)
	}

	var problem problemDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Title != expectedErrorTitle || problem.Detail != expectedErrorTitle {
		t.Fatalf("expected fallback title/detail to be Error, got %+v", problem)
	}

	if got := parameterValidationDetail(nil); got != parameterValidationFailed {
		t.Fatalf("expected default validation detail, got %q", got)
	}
	if got := parameterValidationDetail([]datasetapi.ParameterError{{Message: "bad value"}}); got != "bad value" {
		t.Fatalf("expected message-only validation detail, got %q", got)
	}
	if got := parameterValidationDetail([]datasetapi.ParameterError{{}}); got != parameterValidationFailed {
		t.Fatalf("expected fallback validation detail, got %q", got)
	}
	if got := parameterValidationDetail([]datasetapi.ParameterError{
		{Name: "limit", Message: "too large"},
		{Message: "bad stage"},
		{Name: "scope"},
	}); got != "parameter limit: too large; bad stage; parameter scope: invalid value" {
		t.Fatalf("expected aggregated validation detail, got %q", got)
	}
}

func TestHTTPMetricsHandleNilAndAlreadyRegisteredCollectors(t *testing.T) {
	nilMetrics, err := NewHTTPMetrics(nil)
	if err != nil {
		t.Fatalf("new metrics without registerer: %v", err)
	}
	nilMetrics.Observe(http.MethodPost, datasetTemplatesPath, http.StatusCreated, time.Millisecond)

	registry := prometheus.NewRegistry()
	first, err := NewHTTPMetrics(registry)
	if err != nil {
		t.Fatalf("new metrics first registry: %v", err)
	}
	second, err := NewHTTPMetrics(registry)
	if err != nil {
		t.Fatalf("new metrics second registry: %v", err)
	}
	if first.requests != second.requests || first.duration != second.duration {
		t.Fatalf("expected already-registered collectors to be reused")
	}

	second.Observe("", "", 0, time.Millisecond)
	if testutil.ToFloat64(first.requests.WithLabelValues(http.MethodGet, "unmatched", "200")) != 1 {
		t.Fatalf("expected observe to use default labels when values are blank")
	}
}

func TestCaptureStatusWriterReusesExistingRecorder(t *testing.T) {
	rec := httptest.NewRecorder()
	first := captureStatusWriter(rec)
	second := captureStatusWriter(first)

	if first != second {
		t.Fatalf("expected existing status writer to be reused")
	}
}
