package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
			"id":        {Type: typeString},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
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
			prop:         definitionSpec{Type: typeString, Format: dateTimeFormat},
			required:     true,
			wantType:     "time.Time",
			wantUsesTime: true,
		},
		{
			name:         "stringOptional",
			prop:         definitionSpec{Type: typeString},
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
			"id":        {Type: typeString},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
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
	openapiPath := filepath.Join(tmpDir, "entity-model.yaml")
	pgSQLPath := filepath.Join(tmpDir, "postgres.sql")
	sqlitePath := filepath.Join(tmpDir, "sqlite.sql")

	content := `{"version":"0.0.1","metadata":{"status":"seed"},"enums":{"kind":{"values":["one"]}},"definitions":{"id":{"type":"string"},"timestamp":{"type":"string","format":"date-time"}},"entities":{"Entity":{"required":["id","created_at","updated_at","kind"],"properties":{"id":{"$ref":"#/definitions/id"},"created_at":{"$ref":"#/definitions/timestamp"},"updated_at":{"$ref":"#/definitions/timestamp"},"kind":{"$ref":"#/enums/kind"},"at":{"type":"string","format":"date-time"}}}}}`
	if err := os.WriteFile(schemaPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	runMainWithArgs(t, []string{"-schema", schemaPath, "-out", outPath, "-openapi", openapiPath, "-sql-postgres", pgSQLPath, "-sql-sqlite", sqlitePath})

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if _, err := os.Stat(openapiPath); err != nil {
		t.Fatalf("expected openapi output file: %v", err)
	}
	for _, path := range []string{pgSQLPath, sqlitePath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected sql output file %s: %v", path, err)
		}
	}
}

func TestMainSkipsOpenAPIWhenFlagMissing(t *testing.T) {
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.json")
	outPath := filepath.Join(tmpDir, "out.go")

	content := `{"version":"0.0.1","metadata":{"status":"seed"},"enums":{"kind":{"values":["one"]}},"definitions":{"id":{"type":"string"}},"entities":{"Entity":{"required":["id"],"properties":{"id":{"$ref":"#/definitions/id"}}}}}`
	if err := os.WriteFile(schemaPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	runMainWithArgs(t, []string{"-schema", schemaPath, "-out", outPath})

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "entity-model.yaml")); err == nil {
		t.Fatalf("did not expect openapi file without flag")
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

func TestBuildOpenAPIDoc(t *testing.T) {
	doc := schemaDoc{
		Version: "0.1.0",
		Enums: map[string]enumSpec{
			"status": {Values: []string{"pending", "approved"}},
		},
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
			"metadata": {
				Type: "object",
				Properties: map[string]json.RawMessage{
					"note": raw(`{"type":"string"}`),
				},
				Required: []string{"note"},
			},
		},
		Entities: map[string]entitySpec{
			"Thing": {
				Required: []string{"id", "created_at", "updated_at", "name", "status"},
				Properties: map[string]json.RawMessage{
					"id":          raw(`{"$ref":"#/definitions/id"}`),
					"created_at":  raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at":  raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":        raw(`{"type":"string"}`),
					"status":      raw(`{"$ref":"#/enums/status"}`),
					"tags":        raw(`{"type":"array","items":{"type":"string"}}`),
					"metadata":    raw(`{"$ref":"#/definitions/metadata"}`),
					"attributes":  raw(`{"type":"object","additionalProperties":true}`),
					"emptyObject": raw(`{"type":"object","properties":{},"required":[]}`),
				},
			},
		},
	}

	api, err := buildOpenAPIDoc(doc)
	if err != nil {
		t.Fatalf("buildOpenAPIDoc: %v", err)
	}

	components := api["components"].(map[string]any)["schemas"].(map[string]any)
	thing := components["Thing"].(map[string]any)
	thingProps := thing["properties"].(map[string]any)
	if ro, ok := thingProps["id"].(map[string]any)["readOnly"].(bool); !ok || !ro {
		t.Fatalf("expected id to be readOnly in Thing")
	}
	create := components["ThingCreate"].(map[string]any)
	createProps := create["properties"].(map[string]any)
	if _, ok := createProps["id"]; ok {
		t.Fatalf("id should be omitted from ThingCreate")
	}
	if _, ok := createProps["created_at"]; ok {
		t.Fatalf("created_at should be omitted from ThingCreate")
	}
	if attrs, ok := createProps["attributes"].(map[string]any); !ok || attrs["additionalProperties"] != true {
		t.Fatalf("expected attributes to allow additionalProperties")
	}
	update := components["ThingUpdate"].(map[string]any)
	if update["required"] != nil {
		t.Fatalf("update schema should not set required fields")
	}
	if enumSchema, ok := components["Status"].(map[string]any); ok {
		if enumType := enumSchema["type"]; enumType != typeString {
			t.Fatalf("expected enum schema type string, got %v", enumType)
		}
	}
	if metaSchema, ok := components["Metadata"].(map[string]any); ok {
		props := metaSchema["properties"].(map[string]any)
		if _, ok := props["note"]; !ok {
			t.Fatalf("expected metadata.note property in schema")
		}
	}
}

func TestEncodeYAMLDeterministic(t *testing.T) {
	doc := openAPIDoc{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "T",
			"version": "1.0.0",
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"Item": map[string]any{
					"type": "object",
					"required": []any{
						"name",
					},
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
				},
			},
		},
	}

	yaml, err := encodeYAML(doc)
	if err != nil {
		t.Fatalf("encodeYAML: %v", err)
	}

	got := strings.TrimSpace(string(yaml))
	want := strings.TrimSpace("" +
		`components:` + "\n" +
		`  schemas:` + "\n" +
		`    Item:` + "\n" +
		`      properties:` + "\n" +
		`        name:` + "\n" +
		`          type: "string"` + "\n" +
		`        tags:` + "\n" +
		`          items:` + "\n" +
		`            type: "string"` + "\n" +
		`          type: "array"` + "\n" +
		`      required:` + "\n" +
		`        - "name"` + "\n" +
		`      type: "object"` + "\n" +
		`info:` + "\n" +
		`  title: "T"` + "\n" +
		`  version: "1.0.0"` + "\n" +
		`openapi: "3.1.0"` + "\n")

	if got != want {
		t.Fatalf("unexpected YAML output\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestEncodeYAMLScalar(t *testing.T) {
	yaml, err := encodeYAML("value")
	if err != nil {
		t.Fatalf("encodeYAML: %v", err)
	}
	if string(yaml) != "\"value\"\n" {
		t.Fatalf("unexpected scalar encoding: %q", string(yaml))
	}
}

func TestEncodeYAMLSliceRoot(t *testing.T) {
	yaml, err := encodeYAML([]any{"a", map[string]any{"k": "v"}})
	if err != nil {
		t.Fatalf("encodeYAML: %v", err)
	}
	text := string(yaml)
	if !strings.Contains(text, "- \"a\"") || !strings.Contains(text, "k: \"v\"") {
		t.Fatalf("unexpected slice encoding: %s", text)
	}
}

func TestEncodeYAMLMapRoot(t *testing.T) {
	yaml, err := encodeYAML(map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("encodeYAML: %v", err)
	}
	if !strings.Contains(string(yaml), "k: \"v\"") {
		t.Fatalf("unexpected map encoding: %s", string(yaml))
	}
}

func TestEncodeOpenAPIYAMLHeader(t *testing.T) {
	doc := openAPIDoc{"openapi": "3.1.0"}
	out, err := encodeOpenAPIYAML(doc)
	if err != nil {
		t.Fatalf("encodeOpenAPIYAML: %v", err)
	}
	text := string(out)
	if !strings.HasPrefix(text, "# Code generated by internal/tools/entitymodel/generate. DO NOT EDIT.") {
		t.Fatalf("expected generated header, got:\n%s", text)
	}
	if !strings.Contains(text, "openapi: \"3.1.0\"") {
		t.Fatalf("expected openapi field in output:\n%s", text)
	}
}

func TestSQLTypeForPropertyVariants(t *testing.T) {
	enums := map[string]enumSpec{
		"status": {Values: []string{"ready"}},
	}
	defs := map[string]definitionSpec{
		"id":        {Type: typeString, Format: "uuid"},
		"timestamp": {Type: typeString, Format: dateTimeFormat},
		"meta": {
			Type: "object",
			Properties: map[string]json.RawMessage{
				"k": raw(`{"type":"string"}`),
			},
		},
	}

	tests := []struct {
		name string
		prop definitionSpec
		want string
	}{
		{name: "string", prop: definitionSpec{Type: typeString}, want: "TEXT"},
		{name: "uuid", prop: definitionSpec{Type: typeString, Format: "uuid"}, want: "UUID"},
		{name: "enumRef", prop: definitionSpec{Ref: "#/enums/status"}, want: "TEXT"},
		{name: "dateTime", prop: definitionSpec{Ref: "#/definitions/timestamp"}, want: "TIMESTAMPTZ"},
		{name: "int", prop: definitionSpec{Type: typeInteger}, want: "INTEGER"},
		{name: "number", prop: definitionSpec{Type: typeNumber}, want: "DOUBLE PRECISION"},
		{name: "bool", prop: definitionSpec{Type: typeBoolean}, want: "BOOLEAN"},
		{name: "array", prop: definitionSpec{Type: typeArray}, want: "JSONB"},
		{name: "objectWithProps", prop: defs["meta"], want: "JSONB"},
		{name: "unknownFallsBack", prop: definitionSpec{}, want: "JSONB"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := sqlTypeForProperty(tt.prop, enums, defs, postgresDialect)
			if err != nil {
				t.Fatalf("sqlTypeForProperty: %v", err)
			}
			if got != tt.want {
				t.Fatalf("type mismatch: got %s want %s", got, tt.want)
			}
		})
	}
}

func TestResolvePropertyUnknownRef(t *testing.T) {
	if _, err := resolveProperty(definitionSpec{Ref: "#/definitions/missing"}, nil, map[string]definitionSpec{}); err == nil {
		t.Fatalf("expected error for missing ref")
	}
}

func TestBuildSQLForDialectProducesExpectedTables(t *testing.T) {
	doc := schemaDoc{
		Enums: map[string]enumSpec{
			"status": {Values: []string{"active"}},
		},
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
			"entity_id": {Type: typeString, Format: "uuid"},
			"metadata": {
				Type: "object",
				Properties: map[string]json.RawMessage{
					"note": raw(`{"type":"string"}`),
				},
			},
		},
		Entities: map[string]entitySpec{
			"Owner": {
				Required: []string{"id", "created_at", "updated_at", "name"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":       raw(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{},
			},
			"Widget": {
				Required: []string{"id", "created_at", "updated_at", "name", "owner_id"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":       raw(`{"type":"string"}`),
					"owner_id":   raw(`{"$ref":"#/definitions/entity_id"}`),
					"tags":       raw(`{"type":"array","items":{"type":"string"}}`),
					"metadata":   raw(`{"$ref":"#/definitions/metadata"}`),
					"quantity":   raw(`{"type":"number"}`),
				},
				Relationships: map[string]relationshipSpec{
					"owner_id": {Target: "Owner", Cardinality: "1"},
					"tags":     {Target: "Owner", Cardinality: "0..n"},
				},
			},
		},
	}

	pg, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}
	if !strings.Contains(pg, "CREATE TABLE IF NOT EXISTS owners") {
		t.Fatalf("expected owners table, got:\n%s", pg)
	}
	for _, want := range []string{"owner_id UUID", "tags JSONB", "FOREIGN KEY (owner_id) REFERENCES owners(id)"} {
		if !strings.Contains(pg, want) {
			t.Fatalf("expected postgres SQL to contain %q", want)
		}
	}

	sqlite, err := buildSQLForDialect(doc, sqliteDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect sqlite: %v", err)
	}
	for _, want := range []string{"quantity REAL", "TEXT", "FOREIGN KEY (owner_id) REFERENCES owners(id)"} {
		if !strings.Contains(sqlite, want) {
			t.Fatalf("expected sqlite SQL to contain %q", want)
		}
	}
}

func TestBuildSQLForDialectRequiresID(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"timestamp": {Type: typeString, Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"Broken": {
				Required: []string{"created_at"},
				Properties: map[string]json.RawMessage{
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
				},
			},
		},
	}

	if _, err := buildSQLForDialect(doc, postgresDialect); err == nil {
		t.Fatalf("expected missing id column error")
	}
}

func TestToSnakeAndPluralize(t *testing.T) {
	cases := map[string]string{
		"Owner":       "owner",
		"HousingUnit": "housing_unit",
		"APIKey":      "api_key",
		"mixed-Name":  "mixed_name",
		"":            "",
	}
	for input, want := range cases {
		if got := toSnake(input); got != want {
			t.Fatalf("toSnake(%q) = %q, want %q", input, got, want)
		}
	}
	if pluralize("owners") != "owners" {
		t.Fatalf("pluralize should keep trailing s")
	}
	if pluralize("widget") != "widgets" {
		t.Fatalf("pluralize should append s")
	}
	if pluralize("facility") != "facilities" {
		t.Fatalf("pluralize should convert facility to facilities")
	}
}

func TestSchemaForPropertyErrorsOnUnknownRef(t *testing.T) {
	prop := definitionSpec{Ref: "#/definitions/missing"}
	if _, err := schemaForProperty(prop, nil, map[string]definitionSpec{}); err == nil {
		t.Fatalf("expected error for unknown ref")
	}
}

func TestYAMLEncoderBranches(t *testing.T) {
	content := openAPIDoc{
		"emptyMap": map[string]any{},
		"list": []any{
			map[string]any{},
			map[string]any{"k": "v"},
			[]any{},
			[]any{"a", map[string]any{"inner": []any{"x"}}},
		},
		"nothing": []string{},
		"maybe":   nil,
	}

	yaml, err := encodeYAML(content)
	if err != nil {
		t.Fatalf("encodeYAML: %v", err)
	}
	text := string(yaml)
	for _, want := range []string{"emptyMap: {}", "list:", "- {}", "- \"a\"", "inner:", "nothing: []", "maybe: null"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected YAML to contain %q\n%s", want, text)
		}
	}
}

func TestCloneValueDeepCopy(t *testing.T) {
	original := map[string]any{
		"list": []any{map[string]any{"k": "v"}},
	}
	cloned := cloneValue(original).(map[string]any)
	original["list"].([]any)[0].(map[string]any)["k"] = "changed"

	if cloned["list"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected clone to remain unchanged: %#v", cloned)
	}
}

func TestAdditionalPropertiesValueInvalid(t *testing.T) {
	if _, ok := additionalPropertiesValue(raw(`{"not":"bool"}`)); ok {
		t.Fatalf("expected invalid additionalProperties to return false")
	}
}

func TestCloneStringsEmpty(t *testing.T) {
	if out := cloneStrings(nil); out != nil {
		t.Fatalf("expected nil clone for nil input")
	}
}

func TestFormatScalarVariants(t *testing.T) {
	if got := formatScalar(5); got != "5" {
		t.Fatalf("unexpected int scalar: %s", got)
	}
	if got := formatScalar(true); got != "true" {
		t.Fatalf("unexpected bool scalar: %s", got)
	}
	if got := formatScalar(map[string]string{"k": "v"}); got != `{"k":"v"}` {
		t.Fatalf("unexpected map scalar: %s", got)
	}
}

func TestGenerateOpenAPIRejectsMissingRef(t *testing.T) {
	doc := schemaDoc{
		Version:  "0.1.0",
		Metadata: metadataSpec{Status: "seed"},
		Enums: map[string]enumSpec{
			"status": {Values: []string{"ok"}},
		},
		Definitions: map[string]definitionSpec{
			"id": {Type: typeString},
		},
		Entities: map[string]entitySpec{
			"Broken": {
				Required: []string{"id", "ref"},
				Properties: map[string]json.RawMessage{
					"id":  raw(`{"$ref":"#/definitions/id"}`),
					"ref": raw(`{"$ref":"#/definitions/missing"}`),
				},
			},
		},
	}

	if _, err := generateOpenAPI(doc); err == nil {
		t.Fatalf("expected error for missing ref in entity")
	}
}

func TestExitErrUsesExitFunc(t *testing.T) {
	called := 0
	exitFunc = func(code int) {
		called = code
	}
	t.Cleanup(func() {
		exitFunc = os.Exit
	})

	exitErr(errors.New("fail"))
	if called != 1 {
		t.Fatalf("expected exit func to be called with 1, got %d", called)
	}
	exitErr(nil)
	if called != 1 {
		t.Fatalf("exitErr should ignore nil error")
	}
}

func TestWriteFileEmptyPath(t *testing.T) {
	if err := writeFile("", []byte("x")); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestMainExitsOnInvalidSchema(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.go")

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"generate", "-schema", filepath.Join(tmpDir, "missing.json"), "-out", outPath}

	exitFunc = func(code int) {
		panic(fmt.Sprintf("exit:%d", code))
	}
	defer func() {
		exitFunc = os.Exit
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "exit:1") {
			t.Fatalf("expected exit panic, got %v", r)
		}
	}()

	main()
}

func runMainWithArgs(t *testing.T, args []string) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	origArgs := os.Args
	os.Args = append([]string{"generate"}, args...)
	t.Cleanup(func() {
		os.Args = origArgs
	})
	main()
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
