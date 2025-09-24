package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDatasetTemplateBindErrorVariants(t *testing.T) { // renamed to avoid collision
	// nil binder
	var tmpl DatasetTemplate
	if err := tmpl.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected error for nil binder")
	}
	// binder returns error
	tmpl = DatasetTemplate{Key: "k", Version: "v1", Title: "t", Dialect: DatasetDialectSQL, Query: "select 1", Columns: []DatasetColumn{{Name: "c", Type: "string"}}, OutputFormats: []DatasetFormat{FormatJSON}, Binder: func(DatasetEnvironment) (DatasetRunner, error) {
		return nil, errors.New("boom")
	}}
	if err := tmpl.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected binder error")
	}
	// binder returns nil runner
	tmpl.Binder = func(DatasetEnvironment) (DatasetRunner, error) { return nil, nil }
	if err := tmpl.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected error for nil runner")
	}
}

func TestDatasetTemplateBindAndRun2(t *testing.T) { // renamed to avoid collision
	called := false
	tmpl := DatasetTemplate{Key: "k", Version: "v1", Title: "t", Dialect: DatasetDialectSQL, Query: "select 1", Columns: []DatasetColumn{{Name: "c", Type: "string"}}, OutputFormats: []DatasetFormat{FormatJSON}, Binder: func(DatasetEnvironment) (DatasetRunner, error) {
		return func(ctx context.Context, req DatasetRunRequest) (DatasetRunResult, error) {
			called = true
			return DatasetRunResult{Rows: []map[string]any{{"c": 1}}, GeneratedAt: time.Now()}, nil
		}, nil
	}}
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

func TestDatasetTemplateCollectionLessBranches(t *testing.T) { // renamed
	coll := DatasetTemplateCollection{
		{Plugin: "p", Key: "a", Version: "2"},
		{Plugin: "p", Key: "a", Version: "1"},
		{Plugin: "p", Key: "b", Version: "1"},
		{Plugin: "q", Key: "a", Version: "1"},
	}
	if !coll.Less(0, 2) && !coll.Less(1, 0) && !coll.Less(0, 3) {
		if !coll.Less(1, 0) {
			t.Fatalf("Less comparator did not behave as expected")
		}
	}
}
