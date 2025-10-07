package core

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

func TestNewDatasetTemplateFromAPI(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	apiTemplate := datasetapi.Template{
		Key:         "demo",
		Version:     "1.0.0",
		Title:       "Demo",
		Description: "demo",
		Dialect:     datasetapi.DialectSQL,
		Query:       "SELECT 1",
		Parameters: []datasetapi.Parameter{{
			Name:        "stage",
			Type:        "string",
			Enum:        []string{"adult"},
			Description: "stage filter",
		}},
		Columns: []datasetapi.Column{{
			Name:        "value",
			Type:        "string",
			Description: "value column",
		}},
		Metadata: datasetapi.Metadata{
			Source:          "tests",
			Documentation:   "docs",
			RefreshInterval: "PT1H",
			Tags:            []string{"tag"},
			Annotations:     map[string]string{"key": "val"},
		},
		OutputFormats: []datasetapi.Format{datasetapi.FormatJSON, datasetapi.FormatCSV},
	}

	apiTemplate.Binder = func(env datasetapi.Environment) (datasetapi.Runner, error) {
		if env.Now == nil {
			t.Fatalf("expected env.Now to be propagated")
		}
		return func(_ context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
			if req.Template.Key != "demo" {
				t.Fatalf("unexpected template key: %s", req.Template.Key)
			}
			if req.Scope.Requestor != "analyst" {
				t.Fatalf("unexpected requestor: %s", req.Scope.Requestor)
			}
			return datasetapi.RunResult{
				Schema:      req.Template.Columns,
				Rows:        []datasetapi.Row{{"value": 7}},
				Metadata:    map[string]any{"note": "ok"},
				GeneratedAt: env.Now(),
				Format:      datasetapi.FormatCSV,
			}, nil
		}, nil
	}

	converted, err := newDatasetTemplateFromAPI(apiTemplate)
	if err != nil {
		t.Fatalf("newDatasetTemplateFromAPI: %v", err)
	}

	apiTemplate.Parameters[0].Enum[0] = testLiteralMutated
	if converted.Parameters[0].Enum[0] != "adult" {
		t.Fatalf("expected defensive copy of enum")
	}

	converted.Plugin = "frog"
	env := DatasetEnvironment{Now: func() time.Time { return now }}
	if err := converted.bind(env); err != nil {
		t.Fatalf("bind converted template: %v", err)
	}

	params := map[string]any{"stage": "adult"}
	scope := DatasetScope{Requestor: "analyst", ProjectIDs: []string{"project"}}
	result, paramErrs, err := converted.Run(context.Background(), params, scope, FormatJSON)
	if err != nil {
		t.Fatalf("run converted template: %v", err)
	}
	if len(paramErrs) != 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if result.Format != FormatJSON {
		t.Fatalf("expected format override to JSON, got %s", result.Format)
	}
	if len(result.Rows) != 1 || result.Rows[0]["value"].(int) != 7 {
		t.Fatalf("unexpected rows: %+v", result.Rows)
	}
	if result.GeneratedAt != now {
		t.Fatalf("expected generatedAt from env.Now")
	}
}

func TestNewDatasetTemplateFromAPIValidation(t *testing.T) {
	_, err := newDatasetTemplateFromAPI(datasetapi.Template{})
	if err == nil {
		t.Fatalf("expected validation error for empty template")
	}
}

func TestDatasetTemplateRuntimeFacade(t *testing.T) {
	template := DatasetTemplate{
		Plugin: "frog",
		Template: datasetapi.Template{
			Key:           "facade",
			Version:       "1.0.0",
			Title:         "Facade",
			Dialect:       datasetapi.DialectSQL,
			Query:         "SELECT 1",
			Parameters:    []datasetapi.Parameter{{Name: "limit", Type: "integer", Required: true}},
			Columns:       []datasetapi.Column{{Name: "value", Type: "integer"}},
			OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		},
	}
	var capturedScope DatasetScope
	template.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(_ context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
			capturedScope = req.Scope
			return datasetapi.RunResult{
				Schema:      req.Template.Columns,
				Rows:        []datasetapi.Row{{"value": 42}},
				GeneratedAt: time.Unix(0, 0).UTC(),
				Format:      datasetapi.FormatJSON,
			}, nil
		}, nil
	}
	if err := template.bind(DatasetEnvironment{}); err != nil {
		t.Fatalf("bind template: %v", err)
	}

	runtime := newDatasetTemplateRuntime(template)
	if runtime == nil {
		t.Fatalf("expected runtime after binding")
	}
	if runtime.Descriptor().Slug == "" {
		t.Fatalf("expected descriptor slug")
	}
	if !runtime.SupportsFormat(datasetapi.FormatJSON) {
		t.Fatalf("expected JSON support")
	}
	if runtime.SupportsFormat(datasetapi.FormatCSV) {
		t.Fatalf("did not expect CSV support")
	}

	if _, errs := runtime.ValidateParameters(map[string]any{}); len(errs) == 0 {
		t.Fatalf("expected validation error for missing required parameter")
	}
	cleaned, errs := runtime.ValidateParameters(map[string]any{"limit": 1})
	if len(errs) != 0 {
		t.Fatalf("unexpected validation error: %+v", errs)
	}
	if cleaned["limit"].(int) != 1 {
		t.Fatalf("expected cleaned parameter value")
	}

	scope := datasetapi.Scope{Requestor: "analyst", ProjectIDs: []string{"project"}}
	result, paramErrs, err := runtime.Run(context.Background(), map[string]any{"limit": 1}, scope, datasetapi.FormatJSON)
	if err != nil {
		t.Fatalf("runtime run: %v", err)
	}
	if len(paramErrs) != 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if capturedScope.Requestor != "analyst" || len(capturedScope.ProjectIDs) != 1 {
		t.Fatalf("unexpected captured scope: %+v", capturedScope)
	}
	if len(result.Rows) != 1 || result.Rows[0]["value"].(int) != 42 {
		t.Fatalf("unexpected rows: %+v", result.Rows)
	}
}
