package core

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	testPluginFrog     = "frog"
	testLiteralMutated = "mutated"
	testLiteralChanged = "changed"
)

type stringer struct{}

func (stringer) String() string { return testPluginFrog }

func TestDatasetTemplateRunSuccess(t *testing.T) {
	reference := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	invocations := 0
	template := DatasetTemplate{
		Plugin:      testPluginFrog,
		Key:         "snapshot",
		Version:     "1.0.0",
		Title:       "Snapshot",
		Description: "demo",
		Dialect:     DatasetDialectSQL,
		Query:       "SELECT 1",
		Parameters: []DatasetParameter{{
			Name:     "limit",
			Type:     "integer",
			Required: true,
		}},
		Columns: []DatasetColumn{{
			Name: testAttributeOriginalValue,
			Type: "integer",
		}},
		Metadata: DatasetTemplateMetadata{Tags: []string{"demo"}},
		OutputFormats: []DatasetFormat{
			FormatJSON,
			FormatCSV,
		},
		Binder: func(env DatasetEnvironment) (DatasetRunner, error) {
			return func(_ context.Context, req DatasetRunRequest) (DatasetRunResult, error) {
				invocations++
				if req.Parameters["limit"].(int) != 25 {
					t.Fatalf("expected coerced integer parameter, got %v", req.Parameters["limit"])
				}
				return DatasetRunResult{
					Rows:        []map[string]any{{testAttributeOriginalValue: 99}},
					GeneratedAt: env.Now(),
				}, nil
			}, nil
		},
	}

	env := DatasetEnvironment{Now: func() time.Time { return reference }}
	if err := template.bind(env); err != nil {
		t.Fatalf("bind template: %v", err)
	}

	params := map[string]any{"limit": "25"}
	scope := DatasetScope{Requestor: "analyst"}
	result, paramErrs, err := template.Run(context.Background(), params, scope, FormatCSV)
	if err != nil {
		t.Fatalf("run template: %v", err)
	}
	if len(paramErrs) != 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if result.Format != FormatCSV {
		t.Fatalf("expected format csv, got %s", result.Format)
	}
	if len(result.Schema) != 1 || result.Schema[0].Name != testAttributeOriginalValue {
		t.Fatalf("expected schema fallback from template, got %+v", result.Schema)
	}
	if !result.GeneratedAt.Equal(reference) {
		t.Fatalf("expected generated timestamp %v, got %v", reference, result.GeneratedAt)
	}
	if invocations != 1 {
		t.Fatalf("expected single runner invocation, got %d", invocations)
	}

	if !template.SupportsFormat(FormatJSON) {
		t.Fatalf("expected template to support JSON format")
	}
	if template.SupportsFormat(FormatPNG) {
		t.Fatalf("did not expect template to support PNG format")
	}
}

func TestDatasetTemplateRunParameterErrors(t *testing.T) {
	template := DatasetTemplate{
		Plugin:      testPluginFrog,
		Key:         "snapshot",
		Version:     "1.0.0",
		Title:       "Snapshot",
		Description: "demo",
		Dialect:     DatasetDialectSQL,
		Query:       "SELECT 1",
		Parameters: []DatasetParameter{{
			Name:     "limit",
			Type:     "integer",
			Required: true,
		}},
		Columns:       []DatasetColumn{{Name: testAttributeOriginalValue, Type: "integer"}},
		OutputFormats: []DatasetFormat{FormatJSON},
		Binder: func(DatasetEnvironment) (DatasetRunner, error) {
			return func(context.Context, DatasetRunRequest) (DatasetRunResult, error) {
				return DatasetRunResult{}, nil
			}, nil
		},
	}
	if err := template.bind(DatasetEnvironment{}); err != nil {
		t.Fatalf("bind template: %v", err)
	}

	params := map[string]any{"unexpected": true}
	result, paramErrs, err := template.Run(context.Background(), params, DatasetScope{}, FormatJSON)
	if err != nil {
		t.Fatalf("run template: %v", err)
	}
	if len(paramErrs) != 2 {
		t.Fatalf("expected two parameter errors (missing required + unexpected), got %d", len(paramErrs))
	}
	if result.Rows != nil {
		t.Fatalf("expected no rows when validation fails")
	}
}

func TestDatasetValidateParametersCoercion(t *testing.T) {
	when := time.Date(2023, 5, 6, 7, 8, 9, 0, time.UTC)
	template := DatasetTemplate{
		Parameters: []DatasetParameter{
			{Name: "name", Type: "string", Required: true, Enum: []string{testPluginFrog, "newt"}},
			{Name: "count", Type: "integer"},
			{Name: "ratio", Type: "number"},
			{Name: "flag", Type: "boolean"},
			{Name: "as_of", Type: "timestamp"},
			{Name: "note", Type: "string", Default: "n/a"},
		},
	}

	supplied := map[string]any{
		"NAME":  stringer{},
		"count": int64(7),
		"ratio": "3.14",
		"flag":  "true",
		"as_of": when.Format(time.RFC3339),
	}

	cleaned, errs := template.ValidateParameters(supplied)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if cleaned["name"].(string) != testPluginFrog {
		t.Fatalf("expected enum stringer coercion, got %v", cleaned["name"])
	}
	if cleaned["count"].(int) != 7 {
		t.Fatalf("expected integer coercion, got %v", cleaned["count"])
	}
	if cleaned["ratio"].(float64) != 3.14 {
		t.Fatalf("expected float coercion, got %v", cleaned["ratio"])
	}
	if cleaned["flag"].(bool) != true {
		t.Fatalf("expected boolean coercion")
	}
	if !cleaned["as_of"].(time.Time).Equal(when) {
		t.Fatalf("expected timestamp coercion")
	}
	if cleaned["note"].(string) != "n/a" {
		t.Fatalf("expected default value to apply, got %v", cleaned["note"])
	}

	supplied["NAME"] = "invalid"
	_, errs = template.ValidateParameters(supplied)
	if len(errs) == 0 {
		t.Fatalf("expected enum validation failure")
	}
	if errs[0].Name != "name" {
		t.Fatalf("expected error for name parameter, got %+v", errs)
	}
}

func TestDatasetTemplateCollectionSort(t *testing.T) {
	collection := DatasetTemplateCollection{
		{Plugin: testPluginFrog, Key: "b", Version: "0.1.0"},
		{Plugin: testPluginFrog, Key: "a", Version: "0.2.0"},
		{Plugin: "newt", Key: "a", Version: "0.1.0"},
	}
	collection[0].Slug = testPluginFrog + "/b@0.1.0"
	collection[1].Slug = testPluginFrog + "/a@0.2.0"
	collection[2].Slug = "newt/a@0.1.0"

	sort.Sort(collection)

	if collection[0].Key != "a" || collection[0].Version != "0.2.0" {
		t.Fatalf("unexpected sort order: %+v", collection)
	}
	if collection[1].Key != "b" {
		t.Fatalf("expected frog/b second, got %+v", collection)
	}
}

func TestDatasetTemplateBindErrors(t *testing.T) {
	template := DatasetTemplate{Key: "test"}
	if err := template.validate(); err == nil {
		t.Fatalf("expected validation error for incomplete template")
	}

	template = DatasetTemplate{
		Key:         "test",
		Version:     "1.0.0",
		Title:       "Test",
		Description: "demo",
		Dialect:     DatasetDialectSQL,
		Query:       "SELECT 1",
		Columns:     []DatasetColumn{{Name: testAttributeOriginalValue, Type: "integer"}},
		OutputFormats: []DatasetFormat{
			FormatJSON,
		},
		Binder: func(DatasetEnvironment) (DatasetRunner, error) {
			return nil, nil
		},
	}
	if err := template.bind(DatasetEnvironment{}); err == nil {
		t.Fatalf("expected error when binder returns nil runner")
	}

	template.Binder = func(DatasetEnvironment) (DatasetRunner, error) {
		return nil, fmt.Errorf("binder failure")
	}
	if err := template.bind(DatasetEnvironment{}); err == nil || err.Error() != "binder failure" {
		t.Fatalf("expected binder failure to propagate, got %v", err)
	}
}

func TestDatasetValidateParametersUnknownType(t *testing.T) {
	template := DatasetTemplate{
		Parameters: []DatasetParameter{{Name: "mystery", Type: "uuid"}},
	}
	_, errs := template.ValidateParameters(map[string]any{"mystery": testAttributeOriginalValue})
	if len(errs) == 0 {
		t.Fatalf("expected error for unsupported parameter type")
	}
}

func TestDatasetValidateParametersErrorBranches(t *testing.T) {
	template := DatasetTemplate{
		Parameters: []DatasetParameter{
			{Name: "count", Type: "integer", Required: true},
			{Name: "ratio", Type: "number"},
			{Name: "flag", Type: "boolean"},
			{Name: "as_of", Type: "timestamp"},
		},
	}

	supplied := map[string]any{
		"count": 1.5,
		"ratio": true,
		"flag":  1,
		"as_of": "invalid",
	}

	_, errs := template.ValidateParameters(supplied)
	if len(errs) < 4 {
		t.Fatalf("expected coercion errors, got %+v", errs)
	}
	required := map[string]bool{"count": false, "ratio": false, "flag": false, "as_of": false}
	for _, err := range errs {
		if strings.Contains(err.Message, "expects") {
			required[err.Name] = true
		}
	}
	for name, covered := range required {
		if !covered {
			t.Fatalf("expected error for parameter %s, got %+v", name, errs)
		}
	}
}

func TestDatasetTemplateSlugVariants(t *testing.T) {
	template := DatasetTemplate{Key: "demo", Version: "1.0.0"}
	if slug := template.slug(); slug != "demo@1.0.0" {
		t.Fatalf("unexpected slug without plugin: %s", slug)
	}
	template.Plugin = testPluginFrog
	if slug := template.slug(); slug != testPluginFrog+"/demo@1.0.0" {
		t.Fatalf("unexpected slug with plugin: %s", slug)
	}
}

func TestDatasetTemplateValidateFailures(t *testing.T) {
	base := DatasetTemplate{
		Key:         "demo",
		Version:     "1.0.0",
		Title:       "Demo",
		Description: "demo",
		Dialect:     DatasetDialectSQL,
		Query:       "SELECT 1",
		Columns:     []DatasetColumn{{Name: testAttributeOriginalValue, Type: "integer"}},
		OutputFormats: []DatasetFormat{
			FormatJSON,
		},
		Binder: func(DatasetEnvironment) (DatasetRunner, error) {
			return func(context.Context, DatasetRunRequest) (DatasetRunResult, error) {
				return DatasetRunResult{}, nil
			}, nil
		},
	}
	tests := []struct {
		name string
		mut  func(*DatasetTemplate)
	}{
		{"missing key", func(tmpl *DatasetTemplate) { tmpl.Key = "" }},
		{"missing version", func(tmpl *DatasetTemplate) { tmpl.Version = "" }},
		{"missing title", func(tmpl *DatasetTemplate) { tmpl.Title = "" }},
		{"unsupported dialect", func(tmpl *DatasetTemplate) { tmpl.Dialect = DatasetDialect("graphql") }},
		{"missing query", func(tmpl *DatasetTemplate) { tmpl.Query = "" }},
		{"missing columns", func(tmpl *DatasetTemplate) { tmpl.Columns = nil }},
		{"missing formats", func(tmpl *DatasetTemplate) { tmpl.OutputFormats = nil }},
		{"missing binder", func(tmpl *DatasetTemplate) { tmpl.Binder = nil }},
	}
	for _, tc := range tests {
		tmpl := base
		tc.mut(&tmpl)
		if err := tmpl.validate(); err == nil {
			t.Fatalf("%s: expected validation error", tc.name)
		}
	}
}

func TestDatasetCloneHelpers(t *testing.T) {
	parameters := []DatasetParameter{{Name: "enum", Enum: []string{"a", "b"}}, {Name: "plain"}}
	clonedParams := cloneParameters(parameters)
	clonedParams[0].Enum[0] = testLiteralMutated
	if parameters[0].Enum[0] != "a" {
		t.Fatalf("expected parameter enum to remain unchanged")
	}

	columns := []DatasetColumn{{Name: testAttributeOriginalValue, Type: "string"}}
	clonedColumns := cloneColumns(columns)
	clonedColumns[0].Name = testLiteralChanged
	if columns[0].Name != testAttributeOriginalValue {
		t.Fatalf("expected original column to remain value")
	}
}

func TestEnumErrorBranches(t *testing.T) {
	if err := enumError(nil); err == nil {
		t.Fatalf("expected error when enumeration empty")
	}
	if err := enumError([]string{"a", "b"}); err == nil || !strings.Contains(err.Error(), "a, b") {
		t.Fatalf("expected formatted enumeration error, got %v", err)
	}
}

func TestDatasetTemplateBindError(t *testing.T) {
	template := DatasetTemplate{
		Key:         "bind",
		Version:     "1.0.0",
		Title:       "Bind",
		Description: "demo",
		Dialect:     DatasetDialectSQL,
		Query:       "SELECT 1",
		Columns:     []DatasetColumn{{Name: testAttributeOriginalValue, Type: "integer"}},
		OutputFormats: []DatasetFormat{
			FormatJSON,
		},
		Binder: func(DatasetEnvironment) (DatasetRunner, error) {
			return nil, fmt.Errorf("bind error")
		},
	}
	if err := template.bind(DatasetEnvironment{}); err == nil || !strings.Contains(err.Error(), "bind error") {
		t.Fatalf("expected bind error, got %v", err)
	}
}

func TestCoerceParameterAdditionalCases(t *testing.T) {
	template := DatasetTemplate{
		Parameters: []DatasetParameter{
			{Name: "ratio", Type: "number"},
			{Name: "flag", Type: "boolean"},
			{Name: "as_of", Type: "timestamp"},
		},
	}
	supplied := map[string]any{
		"ratio": float32(1.25),
		"flag":  true,
		"as_of": time.Now().UTC(),
	}
	cleaned, errs := template.ValidateParameters(supplied)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if cleaned["ratio"].(float64) != 1.25 {
		t.Fatalf("expected float conversion, got %v", cleaned["ratio"])
	}
	if cleaned["flag"].(bool) != true {
		t.Fatalf("expected boolean to remain true")
	}
	if cleaned["as_of"].(time.Time).IsZero() {
		t.Fatalf("expected timestamp to remain set")
	}
}
