package core

import (
	"context"
	"strings"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

const (
	testPluginFrog     = "frog"
	testLiteralMutated = "mutated"
	testLiteralChanged = "changed"
)

type stringer struct{}

func (stringer) String() string { return testPluginFrog }

func TestDatasetTemplateRunSuccess(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	reference := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	invocations := 0

	template := DatasetTemplate{
		Plugin: testPluginFrog,
		Template: datasetapi.Template{
			Key:         "snapshot",
			Version:     "1.0.0",
			Title:       "Snapshot",
			Description: "demo",
			Dialect:     dialectProvider.SQL(),
			Query:       "SELECT 1",
			Parameters: []datasetapi.Parameter{{
				Name:     "limit",
				Type:     "integer",
				Required: true,
			}},
			Columns: []datasetapi.Column{{
				Name: "value",
				Type: "integer",
			}},
			Metadata: datasetapi.Metadata{Tags: []string{"demo"}},
			OutputFormats: []datasetapi.Format{
				formatProvider.JSON(),
				formatProvider.CSV(),
			},
		},
	}

	template.Binder = func(env datasetapi.Environment) (datasetapi.Runner, error) {
		return func(_ context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
			invocations++
			if req.Parameters["limit"].(int) != 25 {
				t.Fatalf("expected coerced integer parameter, got %v", req.Parameters["limit"])
			}
			return datasetapi.RunResult{
				Rows:        []datasetapi.Row{{"value": 99}},
				GeneratedAt: env.Now(),
			}, nil
		}, nil
	}

	env := DatasetEnvironment{Now: func() time.Time { return reference }}
	if err := template.bind(env); err != nil {
		t.Fatalf("bind template: %v", err)
	}

	params := map[string]any{"limit": "25"}
	scope := datasetapi.Scope{Requestor: "analyst"}
	result, paramErrs, err := template.Run(context.Background(), params, scope, formatProvider.CSV())
	if err != nil {
		t.Fatalf("run template: %v", err)
	}
	if len(paramErrs) != 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if result.Format != formatProvider.CSV() {
		t.Fatalf("expected format csv, got %s", result.Format)
	}
	if len(result.Schema) != 1 || result.Schema[0].Name != "value" {
		t.Fatalf("expected schema fallback from template, got %+v", result.Schema)
	}
	if !result.GeneratedAt.Equal(reference) {
		t.Fatalf("expected generated timestamp %v, got %v", reference, result.GeneratedAt)
	}
	if invocations != 1 {
		t.Fatalf("expected single runner invocation, got %d", invocations)
	}

	if !template.SupportsFormat(formatProvider.JSON()) {
		t.Fatalf("expected template to support JSON format")
	}
	if template.SupportsFormat(formatProvider.PNG()) {
		t.Fatalf("did not expect template to support PNG format")
	}
}

func TestDatasetTemplateRunParameterErrors(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	template := DatasetTemplate{
		Plugin: testPluginFrog,
		Template: datasetapi.Template{
			Key:           "snapshot",
			Version:       "1.0.0",
			Title:         "Snapshot",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Parameters:    []datasetapi.Parameter{{Name: "limit", Type: "integer", Required: true}},
			Columns:       []datasetapi.Column{{Name: "value", Type: "integer"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}
	template.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
	}
	if err := template.bind(DatasetEnvironment{}); err != nil {
		t.Fatalf("bind template: %v", err)
	}

	params := map[string]any{"unexpected": true}
	result, paramErrs, err := template.Run(context.Background(), params, datasetapi.Scope{}, formatProvider.JSON())
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
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	when := time.Date(2023, 5, 6, 7, 8, 9, 0, time.UTC)
	template := DatasetTemplate{
		Template: datasetapi.Template{
			Key:     "coerce",
			Version: "1.0.0",
			Title:   "Coerce",
			Dialect: dialectProvider.SQL(),
			Query:   "select 1",
			Parameters: []datasetapi.Parameter{
				{Name: "name", Type: "string", Required: true, Enum: []string{testPluginFrog, "newt"}},
				{Name: "count", Type: "integer"},
				{Name: "ratio", Type: "number"},
				{Name: "flag", Type: "boolean"},
				{Name: "as_of", Type: "timestamp"},
				{Name: "note", Type: "string", Default: "n/a"},
			},
			Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}
	template.Binder = func(datasetapi.Environment) (datasetapi.Runner, error) {
		return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
			return datasetapi.RunResult{}, nil
		}, nil
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
	if !cleaned["as_of"].(time.Time).Equal(when.UTC()) {
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

func TestDatasetValidateParametersErrorBranches(t *testing.T) {
	template := newValidationTemplate()
	supplied := map[string]any{
		"count": 1.5,
		"ratio": true,
		"Flag":  1,
		"when":  "invalid",
	}
	_, errs := template.ValidateParameters(supplied)
	if len(errs) < 4 {
		t.Fatalf("expected coercion errors, got %+v", errs)
	}
	required := map[string]bool{"count": false, "ratio": false, "Flag": false, "when": false}
	for _, err := range errs {
		if _, ok := required[err.Name]; ok && strings.Contains(err.Message, "expects") {
			required[err.Name] = true
		}
	}
	for name, covered := range required {
		if !covered {
			t.Fatalf("expected error for %s, got %+v", name, errs)
		}
	}
}

func TestDatasetValidateParametersLeftover(t *testing.T) {
	template := newValidationTemplate()
	supplied := map[string]any{"count": 1, "ratio": 2.5, "FLAG": false, "extra": true}
	cleaned, errs := template.ValidateParameters(supplied)
	if len(errs) == 0 {
		t.Fatalf("expected error for leftover parameter")
	}
	if cleaned["Flag"].(bool) != false {
		t.Fatalf("expected case-insensitive match for flag")
	}
	if cleaned["ratio"].(float64) != 2.5 {
		t.Fatalf("expected ratio to coerce to float, got %v", cleaned["ratio"])
	}
	if cleaned["count"].(int) != 1 {
		t.Fatalf("expected integer coercion, got %v", cleaned["count"])
	}
	prev := ""
	for _, err := range errs {
		if prev != "" && err.Name < prev {
			t.Fatalf("expected sorted error names, got %+v", errs)
		}
		prev = err.Name
	}
}

func TestDatasetTemplateSlugVariants(t *testing.T) {
	template := DatasetTemplate{Template: datasetapi.Template{Key: "demo", Version: "1.0.0"}}
	if slug := template.slug(); slug != "demo@1.0.0" {
		t.Fatalf("unexpected slug without plugin: %s", slug)
	}
	template.Plugin = testPluginFrog
	if slug := template.slug(); slug != testPluginFrog+"/demo@1.0.0" {
		t.Fatalf("unexpected slug with plugin: %s", slug)
	}
}

func TestDatasetTemplateDescriptorAndFormats(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	template := DatasetTemplate{
		Plugin: "demo",
		Template: datasetapi.Template{
			Key:           "demo",
			Version:       "1.0.0",
			Title:         "Demo",
			Description:   "desc",
			Dialect:       datasetapi.GetDialectProvider().SQL(),
			Query:         "SELECT 1",
			Parameters:    []datasetapi.Parameter{{Name: "limit", Type: "integer", Required: true}},
			Columns:       []datasetapi.Column{{Name: "value", Type: "integer"}},
			Metadata:      datasetapi.Metadata{Tags: []string{"tag"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}

	desc := template.Descriptor()
	if desc.Key != demoTemplateKey || desc.Plugin != demoTemplateKey || len(desc.Columns) != 1 || len(desc.Parameters) != 1 {
		t.Fatalf("unexpected descriptor: %+v", desc)
	}
	if desc.Columns[0].Name == "" || desc.Parameters[0].Name == "" {
		t.Fatal("descriptor should clone column/parameter definitions")
	}

	if !template.SupportsFormat(formatProvider.JSON()) {
		t.Fatal("template should support declared format")
	}
	if template.SupportsFormat(formatProvider.CSV()) {
		t.Fatal("template should not support undeclared format")
	}
}

func TestDatasetTemplateValidateFailures(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	base := DatasetTemplate{
		Template: datasetapi.Template{
			Key:           "demo",
			Version:       "1.0.0",
			Title:         "Demo",
			Description:   "demo",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Columns:       []datasetapi.Column{{Name: "value", Type: "integer"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
			Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
				return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
					return datasetapi.RunResult{}, nil
				}, nil
			},
		},
	}
	tests := []struct {
		name string
		mut  func(*DatasetTemplate)
	}{
		{"missing key", func(tmpl *DatasetTemplate) { tmpl.Key = "" }},
		{"missing version", func(tmpl *DatasetTemplate) { tmpl.Version = "" }},
		{"missing title", func(tmpl *DatasetTemplate) { tmpl.Title = "" }},
		{"unsupported dialect", func(tmpl *DatasetTemplate) { tmpl.Dialect = datasetapi.Dialect("graphql") }},
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

func TestDatasetValidateParametersHostError(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	template := DatasetTemplate{
		Template: datasetapi.Template{
			Key:           "demo",
			Version:       "1.0.0",
			Title:         "Demo",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}
	if _, errs := template.ValidateParameters(map[string]any{}); len(errs) == 0 || !strings.Contains(errs[0].Message, "binder required") {
		t.Fatalf("expected binder error, got %+v", errs)
	}
}

func TestDatasetTemplateRunWithoutBind(t *testing.T) {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	template := DatasetTemplate{
		Template: datasetapi.Template{
			Key:           "demo",
			Version:       "1.0.0",
			Title:         "Demo",
			Dialect:       dialectProvider.SQL(),
			Query:         "SELECT 1",
			Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		},
	}
	if _, _, err := template.Run(context.Background(), map[string]any{}, datasetapi.Scope{}, FormatJSON); err == nil || !strings.Contains(err.Error(), "not bound") {
		t.Fatalf("expected not bound error, got %v", err)
	}
}

func TestDatasetSlugTrimsWhitespace(t *testing.T) {
	if slug := datasetSlug(" frog ", " key ", " v1 "); slug != "frog/key@v1" {
		t.Fatalf("unexpected trimmed slug: %s", slug)
	}
}

func newValidationTemplate() DatasetTemplate {
	return DatasetTemplate{
		Template: datasetapi.Template{
			Key:     "validation",
			Version: "1.0.0",
			Title:   "Validation",
			Dialect: dialectProvider.SQL(),
			Query:   "SELECT 1",
			Parameters: []datasetapi.Parameter{
				{Name: "count", Type: "integer", Required: true},
				{Name: "ratio", Type: "number"},
				{Name: "Flag", Type: "boolean"},
				{Name: "when", Type: "timestamp"},
			},
			Columns:       []datasetapi.Column{{Name: "value", Type: "string"}},
			OutputFormats: []datasetapi.Format{formatProvider.JSON()},
			Binder: func(datasetapi.Environment) (datasetapi.Runner, error) {
				return func(context.Context, datasetapi.RunRequest) (datasetapi.RunResult, error) {
					return datasetapi.RunResult{}, nil
				}, nil
			},
		},
	}
}
