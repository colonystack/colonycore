package datasets

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

const publicTemplateKey = "public"

func TestDatasetScopeAndCSVHelpers(t *testing.T) {
	columns := []datasetapi.Column{{Name: "value", Type: "string"}}
	rows := make([]datasetapi.Row, csvProgressSampleRows+5)
	for i := range rows {
		rows[i] = datasetapi.Row{"value": strings.Repeat("x", (i%3)+1)}
	}

	if got := estimateCSVSize(columns, nil); got <= 0 {
		t.Fatalf("expected header-only estimate > 0, got %d", got)
	}
	if got := estimateCSVSize(columns, rows); got <= 0 {
		t.Fatalf("expected sampled estimate > 0, got %d", got)
	}
	if got, err := estimateCSVSectionSize(columns, rows[:2]); err != nil || got <= 0 {
		t.Fatalf("expected section estimate > 0, got %d err=%v", got, err)
	}

	descriptor := datasetapi.TemplateDescriptor{Columns: columns}
	if got := csvColumns(descriptor, datasetapi.RunResult{}); len(got) != len(columns) || got[0].Name != "value" {
		t.Fatalf("expected descriptor column fallback, got %+v", got)
	}
	schema := []datasetapi.Column{{Name: "other", Type: "string"}}
	if got := csvColumns(descriptor, datasetapi.RunResult{Schema: schema}); len(got) != len(schema) || got[0].Name != "other" {
		t.Fatalf("expected schema columns, got %+v", got)
	}

	if !matchesScopeAnnotation("", nil) {
		t.Fatalf("expected empty annotation to allow access")
	}
	if !matchesScopeAnnotation(wildcardScopeValue, []string{"project-1"}) {
		t.Fatalf("expected wildcard annotation to allow scoped project")
	}
	if matchesScopeAnnotation(wildcardScopeValue, nil) {
		t.Fatalf("expected wildcard annotation to reject missing scope")
	}
	if !matchesScopeAnnotation("project-2,project-3", []string{"project-1", "project-3"}) {
		t.Fatalf("expected explicit scope intersection to allow access")
	}
	if matchesScopeAnnotation("project-2", []string{"project-1"}) {
		t.Fatalf("expected explicit scope mismatch to reject access")
	}

	values := parseDelimitedValues("alpha, alpha, beta ,")
	if len(values) != 2 || values[0] != "alpha" || values[1] != "beta" {
		t.Fatalf("unexpected parsed values: %+v", values)
	}
	if got := parseDelimitedValues("   "); got != nil {
		t.Fatalf("expected nil for blank delimited value, got %+v", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/datasets/templates", nil)
	req.Header.Set(datasetScopeRequestorHeader, "analyst")
	req.Header.Set(datasetScopeRolesHeader, "reader, analyst")
	req.Header.Set(datasetScopeProjectIDsHeader, "project-1, project-2")
	req.Header.Set(datasetScopeProtocolIDsHeader, "protocol-1")
	scope := datasetScopeFromRequest(req)
	if scope.Requestor != "analyst" || len(scope.Roles) != 2 || len(scope.ProjectIDs) != 2 || len(scope.ProtocolIDs) != 1 {
		t.Fatalf("unexpected scope from request: %+v", scope)
	}
	if zero := datasetScopeFromRequest(nil); zero.Requestor != "" || len(zero.Roles) != 0 || len(zero.ProjectIDs) != 0 || len(zero.ProtocolIDs) != 0 {
		t.Fatalf("expected zero scope for nil request, got %+v", zero)
	}
}

func TestCountingResponseWriterFlushForwards(t *testing.T) {
	base := &flushTrackingResponseWriter{}
	writer := &countingResponseWriter{ResponseWriter: base}

	writer.Flush()
	if !base.flushed {
		t.Fatalf("expected flush to be forwarded")
	}
	if _, err := writer.Write([]byte("abc")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if writer.bytesWritten != 3 {
		t.Fatalf("expected byte count 3, got %d", writer.bytesWritten)
	}
}

func TestFilterTemplatesByScopeAndNoopLogger(t *testing.T) {
	templates := []datasetapi.TemplateDescriptor{
		{Key: publicTemplateKey},
		{
			Key: "project",
			Metadata: datasetapi.Metadata{
				Annotations: map[string]string{templateRBACProjectsAnnotation: wildcardScopeValue},
			},
		},
		{
			Key: "protocol",
			Metadata: datasetapi.Metadata{
				Annotations: map[string]string{templateRBACProtocolsAnnotation: "protocol-1"},
			},
		},
	}

	filtered := filterTemplatesByScope(templates, datasetapi.Scope{
		ProjectIDs:  []string{"project-1"},
		ProtocolIDs: []string{"protocol-1"},
	})
	if len(filtered) != 3 {
		t.Fatalf("expected all templates to be visible, got %+v", filtered)
	}

	filtered = filterTemplatesByScope(templates, datasetapi.Scope{})
	if len(filtered) != 1 || filtered[0].Key != publicTemplateKey {
		t.Fatalf("expected only public template without scope, got %+v", filtered)
	}

	var logger noopRequestLogger
	logger.Info("info")
	logger.Error("error")
}

func TestStreamCSVWriteFailureSetsErrorTrailer(t *testing.T) {
	writer := &failingStreamResponseWriter{failAtWrite: 2}
	descriptor := datasetapi.TemplateDescriptor{
		Key:     "fixture",
		Columns: []datasetapi.Column{{Name: "value", Type: "string"}},
	}
	result := datasetapi.RunResult{
		Rows: []datasetapi.Row{{"value": "ok"}},
	}

	err := streamCSV(writer, descriptor, result)
	if err == nil {
		t.Fatalf("expected streamCSV error")
	}
	if !strings.Contains(err.Error(), "flush final csv rows") {
		t.Fatalf("expected flush error, got %v", err)
	}
	if got := writer.Header().Get(streamErrorTrailer); got == "" {
		t.Fatalf("expected stream error trailer to be set")
	}
}

func TestHandleRunCSVStreamFailureLogsError(t *testing.T) {
	tpl := buildTemplate()
	runtime := core.DatasetTemplateRuntimeForTests(tpl)
	logger := &recordingRequestLogger{}
	handler := &Handler{Logger: logger}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/frog/fixture/1/run?format=csv", strings.NewReader(`{"parameters":{}}`))
	req.Header.Set("Accept", "text/csv")
	writer := &failingStreamResponseWriter{failAtWrite: 2}

	handler.handleRun(writer, req, runtime)

	if len(logger.errors) == 0 {
		t.Fatalf("expected stream failure to be logged")
	}
	if got := writer.Header().Get(streamErrorTrailer); got == "" {
		t.Fatalf("expected stream error trailer to be set by handleRun")
	}
}

type flushTrackingResponseWriter struct {
	header  http.Header
	flushed bool
}

func (w *flushTrackingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*flushTrackingResponseWriter) WriteHeader(int) {}

func (*flushTrackingResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *flushTrackingResponseWriter) Flush() {
	w.flushed = true
}

type failingStreamResponseWriter struct {
	header      http.Header
	writeCount  int
	failAtWrite int
}

func (w *failingStreamResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*failingStreamResponseWriter) WriteHeader(int) {}

func (w *failingStreamResponseWriter) Write(p []byte) (int, error) {
	w.writeCount++
	if w.writeCount >= w.failAtWrite {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func (*failingStreamResponseWriter) Flush() {}

type recordingRequestLogger struct {
	errors []string
}

func (*recordingRequestLogger) Info(string, ...interface{}) {}

func (l *recordingRequestLogger) Error(msg string, _ ...interface{}) {
	l.errors = append(l.errors, msg)
}
