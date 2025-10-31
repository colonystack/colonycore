package datasets

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestHandleRunParameterErrors(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	desc := datasetapi.TemplateDescriptor{
		Plugin:        "stub",
		Key:           "report",
		Version:       "v1",
		OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		Slug:          "stub/report@v1",
	}
	runtime := &runStubTemplate{
		desc: desc,
		runFn: func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
			return datasetapi.RunResult{}, []datasetapi.ParameterError{{Name: "limit", Message: "too large"}}, nil
		},
	}
	h := &Handler{Catalog: singleTemplateCatalog{runtime: runtime}}

	body := `{"parameters":{"limit":100}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/templates/stub/report/v1/run?format=json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "too large") {
		t.Fatalf("expected parameter error in body, got %s", rec.Body.String())
	}
}

func TestHandleExportCreateSuccessWithDerivedSlug(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	desc := datasetapi.TemplateDescriptor{
		Plugin:        "stub",
		Key:           "metrics",
		Version:       "v2",
		OutputFormats: []datasetapi.Format{formatProvider.JSON(), formatProvider.CSV()},
		Slug:          "stub/metrics@v2",
	}
	runtime := &runStubTemplate{desc: desc}
	scheduler := &recordingScheduler{
		record: ExportRecord{ID: "exp-123", Status: ExportStatusQueued},
	}
	h := &Handler{
		Catalog: singleTemplateCatalog{runtime: runtime},
		Exports: scheduler,
	}

	payload := `{
		"template":{"plugin":"stub","key":"metrics","version":"v2"},
		"formats":["json","csv"],
		"parameters":{"limit":10},
		"scope":{"requestor":"alice","roles":["analyst"]},
		"reason":"quarterly review"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/datasets/exports", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if scheduler.last.TemplateSlug != "stub/metrics@v2" {
		t.Fatalf("expected derived slug, got %s", scheduler.last.TemplateSlug)
	}
	if scheduler.last.RequestedBy != "alice" {
		t.Fatalf("expected requested by fallback to scope requestor, got %s", scheduler.last.RequestedBy)
	}
	if scheduler.last.Reason != "quarterly review" {
		t.Fatalf("expected reason carried through, got %s", scheduler.last.Reason)
	}
	if len(scheduler.last.Formats) != 2 || scheduler.last.Formats[0] != formatProvider.JSON() || scheduler.last.Formats[1] != formatProvider.CSV() {
		t.Fatalf("unexpected formats: %+v", scheduler.last.Formats)
	}
	if scheduler.record.ID == "" {
		t.Fatalf("expected scheduler to return record id")
	}
}

type runStubTemplate struct {
	desc       datasetapi.TemplateDescriptor
	validateFn func(map[string]any) (map[string]any, []datasetapi.ParameterError)
	runFn      func(context.Context, map[string]any, datasetapi.Scope, datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error)
}

func (s *runStubTemplate) Descriptor() datasetapi.TemplateDescriptor { return s.desc }
func (s *runStubTemplate) SupportsFormat(format datasetapi.Format) bool {
	for _, f := range s.desc.OutputFormats {
		if f == format {
			return true
		}
	}
	return false
}
func (s *runStubTemplate) ValidateParameters(params map[string]any) (map[string]any, []datasetapi.ParameterError) {
	if s.validateFn != nil {
		return s.validateFn(params)
	}
	return params, nil
}
func (s *runStubTemplate) Run(ctx context.Context, params map[string]any, scope datasetapi.Scope, format datasetapi.Format) (datasetapi.RunResult, []datasetapi.ParameterError, error) {
	if s.runFn != nil {
		return s.runFn(ctx, params, scope, format)
	}
	return datasetapi.RunResult{}, nil, nil
}

type singleTemplateCatalog struct {
	runtime datasetapi.TemplateRuntime
}

func (c singleTemplateCatalog) DatasetTemplates() []datasetapi.TemplateDescriptor {
	return []datasetapi.TemplateDescriptor{c.runtime.Descriptor()}
}

func (c singleTemplateCatalog) ResolveDatasetTemplate(slug string) (datasetapi.TemplateRuntime, bool) {
	if c.runtime.Descriptor().Slug == slug {
		return c.runtime, true
	}
	return nil, false
}

type recordingScheduler struct {
	last   ExportInput
	record ExportRecord
	err    error
}

func (s *recordingScheduler) EnqueueExport(_ context.Context, input ExportInput) (ExportRecord, error) {
	s.last = input
	if s.err != nil {
		return ExportRecord{}, s.err
	}
	return s.record, nil
}

func (s *recordingScheduler) GetExport(string) (ExportRecord, bool) {
	return ExportRecord{}, false
}
