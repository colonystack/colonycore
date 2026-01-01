package main

import (
	"strings"
	"testing"
)

func TestRegistryFixtures(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		wantErr    bool
		wantSubstr string
	}{
		{"valid-minimal", "testutil/fixtures/registry/valid/registry-minimal.yaml", false, ""},
		{"valid-full", "testutil/fixtures/registry/valid/registry-full.yaml", false, ""},
		{"valid-multi", "testutil/fixtures/registry/valid/registry-multi.yaml", false, ""},
		{"edge-empty-lists", "testutil/fixtures/registry/edge/registry-empty-lists.yaml", false, ""},
		{"edge-status-header", "testutil/fixtures/registry/edge/registry-status-header.yaml", false, ""},
		{"invalid-type", "testutil/fixtures/registry/invalid/registry-bad-type.yaml", true, "schema validation"},
		{"invalid-status", "testutil/fixtures/registry/invalid/registry-bad-status.yaml", true, "schema validation"},
		{"invalid-missing-title", "testutil/fixtures/registry/invalid/registry-missing-title.yaml", true, "schema validation"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s", tc.path)
				}
				if tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantSubstr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success for %s, got %v", tc.path, err)
			}
		})
	}
}

func TestLoadJSONSchemaSuccess(t *testing.T) {
	schema, err := loadJSONSchema(registrySchemaPath)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	if schema.Type != schemaTypeObject {
		t.Fatalf("expected root object schema, got %q", schema.Type)
	}
	if _, ok := schema.Properties["documents"]; !ok {
		t.Fatalf("expected documents property in schema")
	}
}

func TestLoadJSONSchemaInvalidJSON(t *testing.T) {
	path := writeTestFile(t, "bad-schema.json", "{not json")
	if _, err := loadJSONSchema(path); err == nil {
		t.Fatalf("expected schema parse error")
	}
}

func TestValidateSchemaErrors(t *testing.T) {
	schema := &jsonSchema{Type: "integer"}
	if err := validateSchema(schema, "$"); err == nil {
		t.Fatalf("expected unsupported type error")
	}
	badEnum := &jsonSchema{Type: schemaTypeArray, Enum: []string{"x"}, Items: &jsonSchema{Type: schemaTypeString}}
	if err := validateSchema(badEnum, "$"); err == nil {
		t.Fatalf("expected enum type error")
	}
	badFormat := &jsonSchema{Type: schemaTypeString, Format: "uuid"}
	if err := validateSchema(badFormat, "$"); err == nil {
		t.Fatalf("expected unsupported format error")
	}
	goodDateTime := &jsonSchema{Type: schemaTypeString, Format: schemaFormatDateTime}
	if err := validateSchema(goodDateTime, "$"); err != nil {
		t.Fatalf("expected date-time format to be allowed, got %v", err)
	}
	goodEmail := &jsonSchema{Type: schemaTypeString, Format: schemaFormatEmail}
	if err := validateSchema(goodEmail, "$"); err != nil {
		t.Fatalf("expected email format to be allowed, got %v", err)
	}
	goodURI := &jsonSchema{Type: schemaTypeString, Format: schemaFormatURI}
	if err := validateSchema(goodURI, "$"); err != nil {
		t.Fatalf("expected uri format to be allowed, got %v", err)
	}
	badPattern := &jsonSchema{Type: schemaTypeString, Pattern: "("}
	if err := validateSchema(badPattern, "$"); err == nil {
		t.Fatalf("expected invalid pattern error")
	}
	goodPattern := &jsonSchema{Type: schemaTypeString, Pattern: "^[1-9][0-9]*$"}
	if err := validateSchema(goodPattern, "$"); err != nil {
		t.Fatalf("unexpected pattern validation error: %v", err)
	}
	if goodPattern.patternRE == nil {
		t.Fatalf("expected compiled pattern")
	}
	badMinItems := &jsonSchema{Type: schemaTypeString, MinItems: intPtr(1)}
	if err := validateSchema(badMinItems, "$"); err == nil {
		t.Fatalf("expected minItems type error")
	}
	badMinLength := &jsonSchema{Type: schemaTypeArray, MinLength: intPtr(1), Items: &jsonSchema{Type: schemaTypeString}}
	if err := validateSchema(badMinLength, "$"); err == nil {
		t.Fatalf("expected minLength type error")
	}
	missingProps := &jsonSchema{Type: schemaTypeObject}
	if err := validateSchema(missingProps, "$"); err == nil {
		t.Fatalf("expected object properties error")
	}
	missingItems := &jsonSchema{Type: schemaTypeArray}
	if err := validateSchema(missingItems, "$"); err == nil {
		t.Fatalf("expected array items error")
	}
	requiredMissing := &jsonSchema{
		Type:       schemaTypeObject,
		Required:   []string{"id"},
		Properties: map[string]*jsonSchema{"name": {Type: schemaTypeString}},
	}
	if err := validateSchema(requiredMissing, "$"); err == nil {
		t.Fatalf("expected required property error")
	}
}

func TestValidateRegistrySchemaErrors(t *testing.T) {
	if err := validateRegistrySchema(nil, &jsonSchema{}); err == nil {
		t.Fatalf("expected nil registry error")
	}
	if err := validateRegistrySchema(&Registry{}, nil); err == nil {
		t.Fatalf("expected nil schema error")
	}
}

func TestValidateValueRules(t *testing.T) {
	schema := &jsonSchema{
		Type:                 schemaTypeObject,
		AdditionalProperties: boolPtr(false),
		Required:             []string{"id", "tags"},
		Properties: map[string]*jsonSchema{
			"id":   {Type: schemaTypeString, MinLength: intPtr(1)},
			"tags": {Type: schemaTypeArray, Items: &jsonSchema{Type: schemaTypeString}, MinItems: intPtr(1)},
		},
	}
	good := map[string]any{"id": "ok", "tags": []any{"tag"}}
	if err := validateValue(good, schema, "$"); err != nil {
		t.Fatalf("expected value to be valid, got %v", err)
	}
	missing := map[string]any{"tags": []any{"tag"}}
	if err := validateValue(missing, schema, "$"); err == nil {
		t.Fatalf("expected missing required error")
	}
	extra := map[string]any{"id": "ok", "tags": []any{"tag"}, "extra": "x"}
	if err := validateValue(extra, schema, "$"); err == nil {
		t.Fatalf("expected additional property error")
	}
	short := map[string]any{"id": "", "tags": []any{"tag"}}
	if err := validateValue(short, schema, "$"); err == nil {
		t.Fatalf("expected minLength error")
	}
	fewTags := map[string]any{"id": "ok", "tags": []any{}}
	if err := validateValue(fewTags, schema, "$"); err == nil {
		t.Fatalf("expected minItems error")
	}
	enumSchema := &jsonSchema{Type: schemaTypeString, Enum: []string{statusMap[statusDraftKey]}}
	if err := validateValue("Accepted", enumSchema, "$.status"); err == nil {
		t.Fatalf("expected enum error")
	}
	dateSchema := &jsonSchema{Type: schemaTypeString, Format: schemaFormatDate}
	if err := validateValue("2025-13-01", dateSchema, "$.date"); err == nil {
		t.Fatalf("expected date format error")
	}
	dateTimeSchema := &jsonSchema{Type: schemaTypeString, Format: schemaFormatDateTime}
	if err := validateValue("2025-01-01T12:30:00Z", dateTimeSchema, "$.updated"); err != nil {
		t.Fatalf("expected date-time to be valid, got %v", err)
	}
	if err := validateValue("not-a-datetime", dateTimeSchema, "$.updated"); err == nil {
		t.Fatalf("expected date-time format error")
	}
	emailSchema := &jsonSchema{Type: schemaTypeString, Format: schemaFormatEmail}
	if err := validateValue("user@example.com", emailSchema, "$.email"); err != nil {
		t.Fatalf("expected email to be valid, got %v", err)
	}
	if err := validateValue("invalid@", emailSchema, "$.email"); err == nil {
		t.Fatalf("expected email format error")
	}
	uriSchema := &jsonSchema{Type: schemaTypeString, Format: schemaFormatURI}
	if err := validateValue("https://example.com/path", uriSchema, "$.uri"); err != nil {
		t.Fatalf("expected uri to be valid, got %v", err)
	}
	if err := validateValue("not a uri", uriSchema, "$.uri"); err == nil {
		t.Fatalf("expected uri format error")
	}
	patternSchema := &jsonSchema{Type: schemaTypeString, Pattern: "^[0-9]+$"}
	if err := validateSchema(patternSchema, "$.quorum"); err != nil {
		t.Fatalf("expected pattern schema to be valid, got %v", err)
	}
	if patternSchema.patternRE == nil {
		t.Fatalf("expected compiled pattern")
	}
	if err := validateValue("nope", patternSchema, "$.quorum"); err == nil {
		t.Fatalf("expected pattern error")
	}
}

func TestDocumentToMapIncludesLists(t *testing.T) {
	doc := Document{ID: "ID", Type: "RFC", Title: "Title", Status: statusMap[statusDraftKey], Path: "docs/rfc/doc.md", Authors: []string{"A"}}
	m, err := documentToMap(doc)
	if err != nil {
		t.Fatalf("documentToMap error: %v", err)
	}
	if got, ok := m["id"].(string); !ok || got != doc.ID {
		t.Fatalf("expected id %q, got %v", doc.ID, m["id"])
	}
	if got, ok := m["status"].(string); !ok || got != doc.Status {
		t.Fatalf("expected status %q, got %v", doc.Status, m["status"])
	}
	if _, ok := m["authors"].([]any); !ok {
		t.Fatalf("expected authors to be []any")
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}
