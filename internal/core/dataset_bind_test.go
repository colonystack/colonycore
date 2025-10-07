package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

func TestDatasetTemplateBindErrorVariants(t *testing.T) {
	tmpl := DatasetTemplate{
		Template: datasetapi.Template{
			Key:           "k",
			Version:       "v1",
			Title:         "t",
			Dialect:       datasetapi.DialectSQL,
			Query:         "select 1",
			Columns:       []datasetapi.Column{{Name: "c", Type: "string"}},
			OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		},
	}
	if err := tmpl.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected error for nil binder")
	}

	tmpl.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) {
		return nil, errors.New("boom")
	}
	if err := tmpl.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected binder error")
	}

	tmpl.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) { return nil, nil }
	if err := tmpl.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected error for nil runner")
	}
}

func TestDatasetTemplateBindAndRun(t *testing.T) {
	called := false
	tmpl := DatasetTemplate{
		Template: datasetapi.Template{
			Key:           "k",
			Version:       "v1",
			Title:         "t",
			Dialect:       datasetapi.DialectSQL,
			Query:         "select 1",
			Columns:       []datasetapi.Column{{Name: "c", Type: "string"}},
			OutputFormats: []datasetapi.Format{datasetapi.FormatJSON},
		},
	}
	tmpl.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			called = true
			return datasetapi.RunResult{Rows: []datasetapi.Row{{"c": 1}}, GeneratedAt: time.Now().UTC(), Format: datasetapi.FormatJSON}, nil
		}, nil
	}
	if err := tmpl.bind(DatasetEnvironment{}); err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	res, paramErrs, err := tmpl.Run(context.Background(), map[string]any{}, DatasetScope{}, FormatJSON)
	if err != nil || len(paramErrs) != 0 {
		t.Fatalf("run unexpected errors: %v %v", err, paramErrs)
	}
	if !called {
		t.Fatalf("runner not invoked")
	}
	if res.Format != FormatJSON {
		t.Fatalf("expected format json, got %s", res.Format)
	}
	if len(res.Schema) == 0 || res.Schema[0].Name != "c" {
		t.Fatalf("expected default schema clone")
	}
}

func TestDatasetSortTemplateDescriptors(t *testing.T) {
	descriptors := []datasetapi.TemplateDescriptor{
		{Plugin: "p", Key: "a", Version: "2"},
		{Plugin: "p", Key: "a", Version: "1"},
		{Plugin: "p", Key: "b", Version: "1"},
		{Plugin: "q", Key: "a", Version: "1"},
	}
	descriptors[0].Slug = "p/a@2"
	descriptors[1].Slug = "p/a@1"
	descriptors[2].Slug = "p/b@1"
	descriptors[3].Slug = "q/a@1"

	datasetapi.SortTemplateDescriptors(descriptors)

	if descriptors[0].Key != "a" || descriptors[0].Version != "1" {
		t.Fatalf("unexpected sort order: %+v", descriptors)
	}
	if descriptors[1].Key != "a" || descriptors[1].Version != "2" {
		t.Fatalf("expected frog/a@2 second, got %+v", descriptors)
	}
}
