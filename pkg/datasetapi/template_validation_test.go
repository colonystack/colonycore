package datasetapi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateTemplateSuccess(t *testing.T) {
	tpl := validTemplateForValidation()
	if err := ValidateTemplate(tpl); err != nil {
		t.Fatalf("ValidateTemplate: %v", err)
	}
}

func TestValidateTemplateFailures(t *testing.T) {
	formatProvider := GetFormatProvider()

	cases := []struct {
		name         string
		mut          func(*Template)
		expectFields []string
	}{
		{
			name: "missing required fields",
			mut: func(tpl *Template) {
				tpl.Key = ""
				tpl.Version = ""
				tpl.Title = ""
				tpl.Query = ""
				tpl.OutputFormats = nil
				tpl.Columns = nil
				tpl.Binder = nil
			},
			expectFields: []string{"key", "version", "title", "query", "output_formats", "columns", "binder"},
		},
		{
			name: "sql mutating statement",
			mut: func(tpl *Template) {
				tpl.Query = "UPDATE organisms SET name = 'x'"
			},
			expectFields: []string{"query"},
		},
		{
			name: "dsl missing report and select",
			mut: func(tpl *Template) {
				dialectProvider := GetDialectProvider()
				tpl.Dialect = dialectProvider.DSL()
				tpl.Query = "SHOW frogs"
			},
			expectFields: []string{"query"},
		},
		{
			name: "duplicate parameter names",
			mut: func(tpl *Template) {
				tpl.Parameters = append(tpl.Parameters, Parameter{Name: "limit", Type: "integer"})
			},
			expectFields: []string{"parameters[1].name"},
		},
		{
			name: "invalid default and example",
			mut: func(tpl *Template) {
				tpl.Parameters = []Parameter{{
					Name:    "stage",
					Type:    "string",
					Example: json.RawMessage("not-json"),
					Default: json.RawMessage(`{"invalid":"shape"}`),
				}}
			},
			expectFields: []string{"parameters[0].example", "parameters[0].default"},
		},
		{
			name: "duplicate columns",
			mut: func(tpl *Template) {
				tpl.Columns = []Column{{Name: "value", Type: "integer"}, {Name: "VALUE", Type: "integer"}}
			},
			expectFields: []string{"columns[1].name"},
		},
		{
			name: "unsupported and duplicate formats",
			mut: func(tpl *Template) {
				tpl.OutputFormats = []Format{formatProvider.JSON(), Format("yaml"), formatProvider.JSON()}
			},
			expectFields: []string{"output_formats[1]", "output_formats[2]"},
		},
		{
			name: "invalid metadata major",
			mut: func(tpl *Template) {
				major := 0
				tpl.Metadata.EntityModelMajor = &major
			},
			expectFields: []string{"metadata.entity_model_major"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tpl := validTemplateForValidation()
			tc.mut(&tpl)
			err := ValidateTemplate(tpl)
			if err == nil {
				t.Fatalf("expected validation error")
			}
			var validationErr *TemplateValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected TemplateValidationError, got %T", err)
			}
			for _, field := range tc.expectFields {
				if !containsValidationField(validationErr.Issues, field) {
					t.Fatalf("expected issue for %s, got %+v", field, validationErr.Issues)
				}
			}
		})
	}
}

func TestValidateTemplateDescriptor(t *testing.T) {
	descriptor := validTemplateDescriptorForValidation()
	if err := ValidateTemplateDescriptor(descriptor); err != nil {
		t.Fatalf("ValidateTemplateDescriptor: %v", err)
	}

	descriptor.Slug = "frog/wrong@0.1.0"
	err := ValidateTemplateDescriptor(descriptor)
	if err == nil {
		t.Fatalf("expected slug mismatch error")
	}
	var validationErr *TemplateValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected TemplateValidationError, got %T", err)
	}
	if !containsValidationField(validationErr.Issues, "slug") {
		t.Fatalf("expected slug issue, got %+v", validationErr.Issues)
	}
	if !strings.Contains(err.Error(), "slug") {
		t.Fatalf("expected slug detail in error string: %v", err)
	}
}

func TestValidateTemplateSQLMutationKeywordInsideLiterals(t *testing.T) {
	tpl := validTemplateForValidation()
	tpl.Query = `SELECT 'update statement' AS action_label, "delete column" AS alias_name FROM organisms`
	if err := ValidateTemplate(tpl); err != nil {
		t.Fatalf("expected string-literal keywords to be ignored, got %v", err)
	}
}

func TestValidateTemplateSQLMutationKeywordInsideComments(t *testing.T) {
	tests := []string{
		"SELECT organism_id -- update organisms\nFROM organisms",
		"SELECT organism_id /* delete organisms */ FROM organisms",
	}
	for _, query := range tests {
		tpl := validTemplateForValidation()
		tpl.Query = query
		if err := ValidateTemplate(tpl); err != nil {
			t.Fatalf("expected comment keywords to be ignored for query %q, got %v", query, err)
		}
	}
}

func validTemplateForValidation() Template {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()
	return Template{
		Key:         "frog_population_snapshot",
		Version:     "0.1.0",
		Title:       "Frog Population Snapshot",
		Description: "Reference template",
		Dialect:     dialectProvider.SQL(),
		Query:       "SELECT organism_id, updated_at FROM organisms",
		Parameters: []Parameter{{
			Name:     "limit",
			Type:     "integer",
			Required: true,
			Default:  json.RawMessage("100"),
		}},
		Columns:       []Column{{Name: "organism_id", Type: "string"}, {Name: "updated_at", Type: "timestamp"}},
		OutputFormats: []Format{formatProvider.JSON(), formatProvider.CSV()},
		Metadata: Metadata{
			Source: "core.organisms",
		},
		Binder: func(Environment) (Runner, error) {
			return func(_ context.Context, _ RunRequest) (RunResult, error) { return RunResult{}, nil }, nil
		},
	}
}

func validTemplateDescriptorForValidation() TemplateDescriptor {
	dialectProvider := GetDialectProvider()
	formatProvider := GetFormatProvider()
	return TemplateDescriptor{
		Plugin:        "frog",
		Key:           "frog_population_snapshot",
		Version:       "0.1.0",
		Title:         "Frog Population Snapshot",
		Description:   "Reference template descriptor",
		Dialect:       dialectProvider.DSL(),
		Query:         "REPORT frog_population_snapshot\nSELECT organism_id FROM organisms",
		Parameters:    []Parameter{{Name: "limit", Type: "integer"}},
		Columns:       []Column{{Name: "organism_id", Type: "string"}},
		OutputFormats: []Format{formatProvider.JSON(), formatProvider.CSV()},
		Slug:          "frog/frog_population_snapshot@0.1.0",
	}
}

func containsValidationField(issues []TemplateValidationIssue, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
