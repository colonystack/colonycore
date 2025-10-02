package core

import (
	"context"
	"testing"

	"colonycore/pkg/datasetapi"
)

func TestPluginRegistryRegisterDatasetTemplateDuplicate(t *testing.T) {
	r := NewPluginRegistry()
	tmpl := datasetapi.Template{Key: "k", Version: "1", Title: "T", Dialect: datasetapi.DialectSQL, Query: "SELECT 1", Columns: []datasetapi.Column{{Name: "c", Type: "string"}}, OutputFormats: []datasetapi.Format{datasetapi.FormatJSON}, Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
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
