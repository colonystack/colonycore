package core

import (
	"context"
	"strings"
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestPluginRegistryRegisterDatasetTemplateDuplicate(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	r := NewPluginRegistry()
	tmpl := datasetapi.Template{Key: "k", Version: "1", Title: "T", Dialect: dialectProvider.SQL(), Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "c", Type: "string"}}, OutputFormats: []datasetapi.Format{formatProvider.JSON()}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(_ context.Context, _ datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
	}}
	if err := r.RegisterDatasetTemplate(tmpl); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := r.RegisterDatasetTemplate(tmpl); err == nil {
		t.Fatalf("expected duplicate registration error")
	}
	if len(r.DatasetTemplates()) == 0 {
		t.Fatalf("expected templates returned")
	}
}

func TestPluginRegistryRegisterDatasetTemplateValidationFailFast(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	r := NewPluginRegistry()
	tmpl := datasetapi.Template{
		Key:     "k",
		Version: "1",
		Title:   "T",
		Dialect: dialectProvider.SQL(),
		Query:   "UPDATE organisms SET name = 'x'",
		Columns: []datasetapi.Column{{Name: "c", Type: "string"}, {Name: "C", Type: "string"}},
		Parameters: []datasetapi.Parameter{
			{Name: "limit", Type: "integer"},
			{Name: "LIMIT", Type: "integer"},
		},
		OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
			return func(_ context.Context, _ datasetapi.RunRequest) (datasetapi.RunResult, error) {
				return datasetapi.RunResult{}, nil
			}, nil
		},
	}
	err := r.RegisterDatasetTemplate(tmpl)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "dataset template validation failed") {
		t.Fatalf("expected validation context, got %v", err)
	}
}
