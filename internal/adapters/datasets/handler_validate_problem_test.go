package datasets

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
)

func TestHandlerValidateInvalidParametersUseProblemJSON(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	template := datasetapi.Template{
		Key:         "validate-problem",
		Version:     "1.0.0",
		Title:       "Validate Problem",
		Description: "validation problem",
		Dialect:     dialectProvider.SQL(),
		Query:       "SELECT 1",
		Parameters: []datasetapi.Parameter{
			{Name: "stage", Type: "string", Enum: []string{"adult", "larva"}},
		},
		Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
		OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{}, nil
			}, nil
		},
	}

	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(testDatasetPlugin{dataset: template}); err != nil {
		t.Fatalf("install plugin: %v", err)
	}

	descriptor := svc.DatasetTemplates()[0]
	handler := NewHandler(svc)
	url := "/api/v1/datasets/templates/" + descriptor.Plugin + "/" + descriptor.Key + "/" + descriptor.Version + "/validate"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"parameters":{"stage":"egg"}}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != problemContentType {
		t.Fatalf("expected problem content type, got %q", got)
	}

	var problem problemDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Type != problemTypeBlank {
		t.Fatalf("expected problem type %q, got %q", problemTypeBlank, problem.Type)
	}
	if problem.Title != http.StatusText(http.StatusUnprocessableEntity) {
		t.Fatalf("expected title %q, got %q", http.StatusText(http.StatusUnprocessableEntity), problem.Title)
	}
	if problem.Status != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, problem.Status)
	}
	if problem.Detail != "parameter stage: value must be one of: adult, larva" {
		t.Fatalf("unexpected problem detail %q", problem.Detail)
	}
	if len(problem.Errors) != 1 {
		t.Fatalf("expected one field error, got %+v", problem.Errors)
	}
	if problem.Errors[0].Name != "stage" {
		t.Fatalf("expected stage field error, got %+v", problem.Errors[0])
	}
	if problem.Errors[0].Message != "value must be one of: adult, larva" {
		t.Fatalf("unexpected field error message %q", problem.Errors[0].Message)
	}
}
