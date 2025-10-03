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
			t.Fatalf("expected now function")
		}
		return func(_ context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
			if req.Template.Key != "demo" {
				t.Fatalf("unexpected template key: %s", req.Template.Key)
			}
			if req.Scope.Requestor != "analyst" {
				t.Fatalf("unexpected requestor: %s", req.Scope.Requestor)
			}
			if len(req.Scope.ProjectIDs) != 1 || req.Scope.ProjectIDs[0] != "project" {
				t.Fatalf("unexpected project scope: %+v", req.Scope.ProjectIDs)
			}
			return datasetapi.RunResult{
				Schema: []datasetapi.Column{{Name: "value", Type: "string"}},
				Rows:   []datasetapi.Row{{"value": 1}},
				Metadata: map[string]any{
					"source": "binder",
				},
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
	scope := DatasetScope{
		Requestor:   "analyst",
		Roles:       []string{"scientist"},
		ProjectIDs:  []string{"project"},
		ProtocolIDs: []string{"protocol"},
	}

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
	if len(result.Rows) != 1 || result.Rows[0]["value"].(int) != 1 {
		t.Fatalf("unexpected rows: %+v", result.Rows)
	}
	if result.Metadata["source"].(string) != "binder" {
		t.Fatalf("unexpected metadata: %+v", result.Metadata)
	}
	if result.GeneratedAt != now {
		t.Fatalf("expected generatedAt from now function")
	}
}

func TestNewDatasetTemplateFromAPIValidation(t *testing.T) {
	_, err := newDatasetTemplateFromAPI(datasetapi.Template{})
	if err == nil {
		t.Fatalf("expected validation error for empty template")
	}
}

func TestAdaptDatasetBinderNilRunner(t *testing.T) {
	template := datasetapi.Template{
		Key:           "demo",
		Version:       "1.0.0",
		Title:         "Demo",
		Description:   "demo",
		Dialect:       datasetapi.DialectSQL,
		Query:         "SELECT 1",
		Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
		OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return nil, nil
		},
	}

	converted, err := newDatasetTemplateFromAPI(template)
	if err != nil {
		t.Fatalf("newDatasetTemplateFromAPI: %v", err)
	}
	if err := converted.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected binder to surface nil runner error")
	}
}
