// Program entitymodelgenerate reads docs/schema/entity-model.json and emits Go projections.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const dateTimeFormat = "date-time"

var exitFunc = os.Exit

type enumSpec struct {
	Values      []string `json:"values"`
	Description string   `json:"description"`
	Initial     string   `json:"initial"`
	Terminal    []string `json:"terminal"`
}

type definitionSpec struct {
	Type                 string                     `json:"type"`
	Format               string                     `json:"format"`
	Ref                  string                     `json:"$ref"`
	Description          string                     `json:"description"`
	Items                *definitionSpec            `json:"items"`
	Properties           map[string]json.RawMessage `json:"properties"`
	Required             []string                   `json:"required"`
	AdditionalProperties json.RawMessage            `json:"additionalProperties"`
}

type stateSpec struct {
	Enum     string   `json:"enum"`
	Initial  string   `json:"initial"`
	Terminal []string `json:"terminal"`
}

type relationshipSpec struct {
	Target      string `json:"target"`
	Cardinality string `json:"cardinality"`
	Storage     string `json:"storage"`
}

type naturalKeySpec struct {
	Fields      []string `json:"fields"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
}

type entitySpec struct {
	Description   string                      `json:"description"`
	NaturalKeys   []naturalKeySpec            `json:"natural_keys"`
	Required      []string                    `json:"required"`
	Properties    map[string]json.RawMessage  `json:"properties"`
	Relationships map[string]relationshipSpec `json:"relationships"`
	States        *stateSpec                  `json:"states"`
	Invariants    []string                    `json:"invariants"`
}

type metadataSpec struct {
	Status string `json:"status"`
}

type idSemanticsSpec struct {
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type schemaDoc struct {
	Version     string                    `json:"version"`
	Metadata    metadataSpec              `json:"metadata"`
	Enums       map[string]enumSpec       `json:"enums"`
	Definitions map[string]definitionSpec `json:"definitions"`
	Entities    map[string]entitySpec     `json:"entities"`
	IDSemantics *idSemanticsSpec          `json:"id_semantics"`
}

func main() {
	schemaPath := flag.String("schema", "docs/schema/entity-model.json", "path to the entity model schema")
	outPath := flag.String("out", "pkg/domain/entitymodel/model_gen.go", "output file for generated Go code")
	openapiPath := flag.String("openapi", "", "output file for generated OpenAPI YAML (optional)")
	sqlPostgresPath := flag.String("sql-postgres", "", "output file for generated Postgres DDL (optional)")
	sqlSQLitePath := flag.String("sql-sqlite", "", "output file for generated SQLite DDL (optional)")
	pluginContractPath := flag.String("plugin-contract", "", "output file for generated plugin contract (optional)")
	flag.Parse()

	doc, err := loadSchema(*schemaPath)
	if err != nil {
		exitErr(err)
	}

	code, err := generateCode(doc)
	if err != nil {
		exitErr(err)
	}

	if err := writeFile(*outPath, code); err != nil {
		exitErr(err)
	}

	if openapiPath != nil && strings.TrimSpace(*openapiPath) != "" {
		openapi, err := generateOpenAPI(doc)
		if err != nil {
			exitErr(err)
		}
		if err := writeFile(*openapiPath, openapi); err != nil {
			exitErr(err)
		}
		fmt.Printf("generated %s from %s\n", *openapiPath, *schemaPath)
	}

	if strings.TrimSpace(*sqlPostgresPath) != "" || strings.TrimSpace(*sqlSQLitePath) != "" {
		pgSQL, sqliteSQL, err := generateSQL(doc)
		if err != nil {
			exitErr(err)
		}
		if path := strings.TrimSpace(*sqlPostgresPath); path != "" {
			if err := writeFile(path, pgSQL); err != nil {
				exitErr(err)
			}
			fmt.Printf("generated %s from %s\n", path, *schemaPath)
		}
		if path := strings.TrimSpace(*sqlSQLitePath); path != "" {
			if err := writeFile(path, sqliteSQL); err != nil {
				exitErr(err)
			}
			fmt.Printf("generated %s from %s\n", path, *schemaPath)
		}
	}

	if path := strings.TrimSpace(*pluginContractPath); path != "" {
		pluginContract, err := generatePluginContract(doc)
		if err != nil {
			exitErr(err)
		}
		if err := writeFile(path, pluginContract); err != nil {
			exitErr(err)
		}
		fmt.Printf("generated %s from %s\n", path, *schemaPath)
	}

	fmt.Printf("generated %s from %s\n", *outPath, *schemaPath)
}

func loadSchema(path string) (schemaDoc, error) {
	//nolint:gosec // generator intentionally reads caller-provided schema path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return schemaDoc{}, fmt.Errorf("read schema: %w", err)
	}

	var doc schemaDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return schemaDoc{}, fmt.Errorf("parse schema: %w", err)
	}

	return doc, nil
}

func generateCode(doc schemaDoc) ([]byte, error) {
	var body strings.Builder
	usesTime := false

	writeEnums(&body, doc.Enums)
	defTime := writeDefinitions(&body, doc.Definitions)
	entityTime, err := writeEntities(&body, doc.Entities, doc.Enums)
	if err != nil {
		return nil, err
	}

	if defTime || entityTime {
		usesTime = true
	}

	var file strings.Builder
	file.WriteString("// Code generated by internal/tools/entitymodel/generate. DO NOT EDIT.\n")
	file.WriteString("package entitymodel\n\n")
	if usesTime {
		file.WriteString("import \"time\"\n\n")
	}
	file.WriteString(body.String())

	formatted, err := format.Source([]byte(file.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated code: %w", err)
	}
	return formatted, nil
}

func writeEnums(body *strings.Builder, enums map[string]enumSpec) {
	names := sortedKeys(enums)
	for _, name := range names {
		enum := enums[name]
		typeName := toCamel(name)
		fmt.Fprintf(body, "// %s enumerates values for %s.\n", typeName, name)
		fmt.Fprintf(body, "type %s string\n\n", typeName)
		body.WriteString("const (\n")
		for _, v := range enum.Values {
			fmt.Fprintf(body, "\t%s%s %s = \"%s\"\n", typeName, toCamel(v), typeName, v)
		}
		body.WriteString(")\n\n")
	}
}

func writeDefinitions(body *strings.Builder, definitions map[string]definitionSpec) bool {
	names := sortedKeys(definitions)
	usesTime := false

	for _, name := range names {
		def := definitions[name]
		if len(def.Properties) == 0 || len(def.Required) == 0 {
			continue
		}

		props, timeUsed := parseProperties(def.Properties)
		if timeUsed {
			usesTime = true
		}
		fmt.Fprintf(body, "// %s is generated from entity-model.json definitions.\n", toCamel(name))
		fmt.Fprintf(body, "type %s struct {\n", toCamel(name))
		for _, propName := range sortedKeys(props) {
			prop := props[propName]
			required := contains(def.Required, propName)
			goType, propUsesTime := goTypeForProperty(prop, required, nil)
			if propUsesTime {
				usesTime = true
			}
			tag := fmt.Sprintf("`json:\"%s", propName)
			if !required {
				tag += ",omitempty"
			}
			tag += "\"`"
			fmt.Fprintf(body, "\t%s %s %s\n", toCamel(propName), goType, tag)
		}
		body.WriteString("}\n\n")
	}

	return usesTime
}

func writeEntities(body *strings.Builder, entities map[string]entitySpec, enums map[string]enumSpec) (bool, error) {
	names := sortedKeys(entities)
	usesTime := false

	for _, name := range names {
		ent := entities[name]
		props, timeUsed := parseProperties(ent.Properties)
		if timeUsed {
			usesTime = true
		}
		fmt.Fprintf(body, "// %s is generated from entity-model.json entities.\n", name)
		fmt.Fprintf(body, "type %s struct {\n", name)
		for _, propName := range sortedKeys(props) {
			prop := props[propName]
			required := contains(ent.Required, propName)
			goType, propUsesTime := goTypeForProperty(prop, required, enums)
			if propUsesTime {
				usesTime = true
			}
			tag := fmt.Sprintf("`json:\"%s", propName)
			if !required {
				tag += ",omitempty"
			}
			tag += "\"`"
			fmt.Fprintf(body, "\t%s %s %s\n", toCamel(propName), goType, tag)
		}
		body.WriteString("}\n\n")
	}

	return usesTime, nil
}

func parseProperties(raw map[string]json.RawMessage) (map[string]definitionSpec, bool) {
	props := make(map[string]definitionSpec, len(raw))
	usesTime := false
	for name, data := range raw {
		var spec definitionSpec
		if err := json.Unmarshal(data, &spec); err == nil {
			props[name] = spec
			if spec.Format == dateTimeFormat || (spec.Items != nil && spec.Items.Format == dateTimeFormat) {
				usesTime = true
			}
		}
	}
	return props, usesTime
}

func goTypeForProperty(prop definitionSpec, required bool, enums map[string]enumSpec) (string, bool) {
	if prop.Ref != "" {
		refType, timeUsed := typeFromRef(prop.Ref, enums)
		return applyOptional(refType, required), timeUsed
	}

	switch prop.Type {
	case "string":
		if prop.Format == dateTimeFormat {
			return applyOptional("time.Time", required), true
		}
		return applyOptional("string", required), false
	case "integer":
		return applyOptional("int", required), false
	case "number":
		return applyOptional("float64", required), false
	case "boolean":
		return applyOptional("bool", required), false
	case "array":
		if prop.Items == nil {
			return "[]any", false
		}
		itemType, timeUsed := goTypeForProperty(*prop.Items, true, enums)
		return "[]" + itemType, timeUsed
	case "object":
		if allowsAdditionalProperties(prop.AdditionalProperties) {
			return "map[string]any", false
		}
		if len(prop.Properties) == 0 {
			return applyOptional("map[string]any", required), false
		}
		// Inline object without a named definition: represent as generic map to keep generator simple.
		return applyOptional("map[string]any", required), false
	}

	return applyOptional("any", required), false
}

func typeFromRef(ref string, enums map[string]enumSpec) (string, bool) {
	if strings.HasPrefix(ref, "#/definitions/") {
		name := strings.TrimPrefix(ref, "#/definitions/")
		switch name {
		case "id", "entity_id":
			return "string", false
		case "timestamp":
			return "time.Time", true
		case "extension_attributes":
			return "map[string]any", false
		default:
			return toCamel(name), false
		}
	}

	if strings.HasPrefix(ref, "#/enums/") {
		name := strings.TrimPrefix(ref, "#/enums/")
		if enums != nil {
			if _, ok := enums[name]; ok {
				return toCamel(name), false
			}
		}
		return toCamel(name), false
	}

	return "any", false
}

func applyOptional(goType string, required bool) string {
	if required {
		return goType
	}

	if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") || strings.HasPrefix(goType, "*") {
		return goType
	}
	return "*" + goType
}

func allowsAdditionalProperties(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var val bool
	if err := json.Unmarshal(raw, &val); err == nil {
		return val
	}
	return false
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeFile(path string, data []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func toCamel(input string) string {
	if input == "" {
		return ""
	}
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	for i, p := range parts {
		parts[i] = applyInitialisms(capitalize(p))
	}
	return strings.Join(parts, "")
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func applyInitialisms(part string) string {
	lower := strings.ToLower(part)
	switch lower {
	case "id":
		return "ID"
	case "ids":
		return "IDs"
	case "api":
		return "API"
	case "url":
		return "URL"
	case "uuid":
		return "UUID"
	case "sku":
		return "SKU"
	default:
		return part
	}
}

func contains(list []string, needle string) bool {
	for _, candidate := range list {
		if strings.EqualFold(candidate, needle) {
			return true
		}
	}
	return false
}

func exitErr(err error) {
	if err == nil {
		return
	}
	//nolint:forbidigo // generator writes to stderr on failure.
	fmt.Fprintln(os.Stderr, err)
	exitFunc(1)
}
