package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenerateMatchesCommitted(t *testing.T) {
	root := repoRoot(t)

	schemaPath := filepath.Join(root, "docs", "schema", "entity-model.json")
	expectedPath := filepath.Join(root, "pkg", "domain", "entitymodel", "model_gen.go")

	doc, err := loadSchema(schemaPath)
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}

	generated, err := generateCode(doc)
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}

	//nolint:gosec // paths are repo-local and deterministic.
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	if !bytes.Equal(bytes.TrimSpace(generated), bytes.TrimSpace(expected)) {
		t.Fatalf("generated code out of date; run `make entity-model-generate`")
	}
}

func TestGenerateCodeIncludesTimeImport(t *testing.T) {
	doc := schemaDoc{
		Enums: map[string]enumSpec{
			"status": {Values: []string{"draft"}},
		},
		Definitions: map[string]definitionSpec{
			"id":        {Type: "string"},
			"timestamp": {Type: "string", Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"Thing": {
				Required: []string{"id", "created_at", "updated_at", "status", "recorded_at"},
				Properties: map[string]json.RawMessage{
					"id":          raw(`{"$ref":"#/definitions/id"}`),
					"created_at":  raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at":  raw(`{"$ref":"#/definitions/timestamp"}`),
					"status":      raw(`{"$ref":"#/enums/status"}`),
					"recorded_at": raw(`{"type":"string","format":"date-time"}`),
				},
			},
		},
	}

	code, err := generateCode(doc)
	if err != nil {
		t.Fatalf("generateCode: %v", err)
	}
	text := string(code)
	if !strings.Contains(text, `import "time"`) {
		t.Fatalf("expected time import in generated code:\n%s", text)
	}
	if !strings.Contains(text, "type Thing struct") || !strings.Contains(text, "Status string") {
		t.Fatalf("expected generated struct and enum:\n%s", text)
	}
}

func TestGoTypeForPropertyVariants(t *testing.T) {
	enums := map[string]enumSpec{
		"status": {Values: []string{"a"}},
	}

	tests := []struct {
		name         string
		prop         definitionSpec
		required     bool
		wantType     string
		wantUsesTime bool
	}{
		{
			name:         "timestampRequired",
			prop:         definitionSpec{Type: "string", Format: dateTimeFormat},
			required:     true,
			wantType:     "time.Time",
			wantUsesTime: true,
		},
		{
			name:         "stringOptional",
			prop:         definitionSpec{Type: "string"},
			required:     false,
			wantType:     "*string",
			wantUsesTime: false,
		},
		{
			name:         "arrayOfInts",
			prop:         definitionSpec{Type: "array", Items: &definitionSpec{Type: "integer"}},
			required:     true,
			wantType:     "[]int",
			wantUsesTime: false,
		},
		{
			name:         "refEnumOptional",
			prop:         definitionSpec{Ref: "#/enums/status"},
			required:     false,
			wantType:     "*Status",
			wantUsesTime: false,
		},
		{
			name:         "refTimestamp",
			prop:         definitionSpec{Ref: "#/definitions/timestamp"},
			required:     true,
			wantType:     "time.Time",
			wantUsesTime: true,
		},
		{
			name:         "objectAdditionalProps",
			prop:         definitionSpec{Type: "object", AdditionalProperties: raw(`true`)},
			required:     true,
			wantType:     "map[string]any",
			wantUsesTime: false,
		},
		{
			name:         "objectAdditionalPropsOptional",
			prop:         definitionSpec{Type: "object", AdditionalProperties: raw(`true`)},
			required:     false,
			wantType:     "map[string]any",
			wantUsesTime: false,
		},
		{
			name:         "refCustom",
			prop:         definitionSpec{Ref: "#/definitions/custom"},
			required:     false,
			wantType:     "*Custom",
			wantUsesTime: false,
		},
		{
			name:         "numberOptional",
			prop:         definitionSpec{Type: "number"},
			required:     false,
			wantType:     "*float64",
			wantUsesTime: false,
		},
		{
			name:         "booleanOptional",
			prop:         definitionSpec{Type: "boolean"},
			required:     false,
			wantType:     "*bool",
			wantUsesTime: false,
		},
		{
			name:         "arrayMissingItems",
			prop:         definitionSpec{Type: "array"},
			required:     true,
			wantType:     "[]any",
			wantUsesTime: false,
		},
		{
			name:         "objectNoProps",
			prop:         definitionSpec{Type: "object"},
			required:     false,
			wantType:     "map[string]any",
			wantUsesTime: false,
		},
		{
			name:         "unknownRef",
			prop:         definitionSpec{Ref: "#/something"},
			required:     false,
			wantType:     "*any",
			wantUsesTime: false,
		},
		{
			name: "inlineObjectProperties",
			prop: definitionSpec{
				Type: "object",
				Properties: map[string]json.RawMessage{
					"field": raw(`{"type":"string"}`),
				},
			},
			required:     false,
			wantType:     "map[string]any",
			wantUsesTime: false,
		},
		{
			name:         "unknownType",
			prop:         definitionSpec{Type: "mystery"},
			required:     false,
			wantType:     "*any",
			wantUsesTime: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotUsesTime := goTypeForProperty(tt.prop, tt.required, enums)
			if gotType != tt.wantType {
				t.Fatalf("type mismatch: got %q want %q", gotType, tt.wantType)
			}
			if gotUsesTime != tt.wantUsesTime {
				t.Fatalf("usesTime mismatch: got %v want %v", gotUsesTime, tt.wantUsesTime)
			}
		})
	}
}

func TestParsePropertiesDetectsTime(t *testing.T) {
	props := map[string]json.RawMessage{
		"ts": raw(`{"type":"string","format":"date-time"}`),
		"n":  raw(`{"type":"number"}`),
	}

	result, usesTime := parseProperties(props)
	if !usesTime {
		t.Fatalf("expected time usage detection")
	}
	if _, ok := result["ts"]; !ok {
		t.Fatalf("missing property ts")
	}
}

func TestDefinitionsGeneration(t *testing.T) {
	doc := schemaDoc{
		Enums: map[string]enumSpec{
			"status": {Values: []string{"ok"}},
		},
		Definitions: map[string]definitionSpec{
			"id":        {Type: "string"},
			"timestamp": {Type: "string", Format: dateTimeFormat},
			"sample_custody_event": {
				Properties: map[string]json.RawMessage{
					"actor":      raw(`{"type":"string"}`),
					"timestamp":  raw(`{"$ref":"#/definitions/timestamp"}`),
					"expires_at": raw(`{"type":"string","format":"date-time"}`),
					"notes":      raw(`{"type":"string"}`),
				},
				Required: []string{"actor", "timestamp", "expires_at"},
				Type:     "object",
			},
		},
		Entities: map[string]entitySpec{},
	}

	code, err := generateCode(doc)
	if err != nil {
		t.Fatalf("generateCode: %v", err)
	}
	text := string(code)
	if !strings.Contains(text, "type SampleCustodyEvent struct") {
		t.Fatalf("expected definitions struct in generated code:\n%s", text)
	}
	if !strings.Contains(text, "*string") {
		t.Fatalf("expected optional field Notes pointer in generated code:\n%s", text)
	}
}

func TestLoadSchema(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "schema-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	content := `{"version":"0.0.0","metadata":{"status":"seed"},"enums":{"x":{"values":["a"]}},"definitions":{},"entities":{}}`
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatalf("Write temp schema: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("Close temp schema: %v", err)
	}

	doc, err := loadSchema(tmp.Name())
	if err != nil {
		t.Fatalf("loadSchema: %v", err)
	}
	if doc.Version != "0.0.0" || doc.Metadata.Status != "seed" {
		t.Fatalf("schema fields not loaded: %+v", doc)
	}
}

func TestLoadSchemaError(t *testing.T) {
	if _, err := loadSchema(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatalf("expected error for missing schema")
	}
}

func TestMainRunsWithTempPaths(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.json")
	outPath := filepath.Join(tmpDir, "out.go")

	content := `{"version":"0.0.1","metadata":{"status":"seed"},"enums":{"kind":{"values":["one"]}},"definitions":{"id":{"type":"string"},"timestamp":{"type":"string","format":"date-time"}},"entities":{"Entity":{"required":["id","created_at","updated_at","kind"],"properties":{"id":{"$ref":"#/definitions/id"},"created_at":{"$ref":"#/definitions/timestamp"},"updated_at":{"$ref":"#/definitions/timestamp"},"kind":{"$ref":"#/enums/kind"},"at":{"type":"string","format":"date-time"}}}}}`
	if err := os.WriteFile(schemaPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	origArgs := os.Args
	defer func() {
		os.Args = origArgs
	}()

	os.Args = []string{"generate", "-schema", schemaPath, "-out", outPath}

	main()

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
}

func TestUtilityHelpers(t *testing.T) {
	if !allowsAdditionalProperties(raw(`true`)) {
		t.Fatalf("expected allowsAdditionalProperties to return true")
	}
	if allowsAdditionalProperties(nil) {
		t.Fatalf("expected allowsAdditionalProperties to return false for nil")
	}
	if allowsAdditionalProperties(raw(`{"not":"bool"}`)) {
		t.Fatalf("expected allowsAdditionalProperties to return false for invalid JSON")
	}

	if got := toCamel("foo_bar-id"); got != "FooBarID" {
		t.Fatalf("toCamel mismatch: %q", got)
	}
	if got := toCamel(""); got != "" {
		t.Fatalf("toCamel empty mismatch: %q", got)
	}
	if got := capitalize("a"); got != "A" {
		t.Fatalf("capitalize single mismatch: %q", got)
	}
	if got := applyInitialisms("api"); got != "API" {
		t.Fatalf("applyInitialisms api mismatch: %q", got)
	}
	if got := applyInitialisms("url"); got != "URL" {
		t.Fatalf("applyInitialisms url mismatch: %q", got)
	}
	if got := applyInitialisms("uuid"); got != "UUID" {
		t.Fatalf("applyInitialisms uuid mismatch: %q", got)
	}
	if got := applyInitialisms("sku"); got != "SKU" {
		t.Fatalf("applyInitialisms sku mismatch: %q", got)
	}
	if !contains([]string{"A", "b"}, "a") {
		t.Fatalf("contains should be case-insensitive")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller for repo root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../.."))
}

func raw(s string) json.RawMessage {
	return json.RawMessage([]byte(s))
}
