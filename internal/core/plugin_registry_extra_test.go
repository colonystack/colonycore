package core

import (
	"context"
	"testing"
)

func TestPluginRegistryRegisterDatasetTemplateDuplicate(t *testing.T) {
	r := NewPluginRegistry()
	tmpl := DatasetTemplate{Key: "k", Version: "1", Title: "T", Dialect: DatasetDialectSQL, Query: "SELECT 1", Columns: []DatasetColumn{{Name: "c", Type: "string"}}, OutputFormats: []DatasetFormat{FormatJSON}, Binder: func(DatasetEnvironment) (DatasetRunner, error) {
		return func(ctx context.Context, req DatasetRunRequest) (DatasetRunResult, error) {
			return DatasetRunResult{}, nil
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
