package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	typeArray   = "array"
	typeBoolean = "boolean"
	typeInteger = "integer"
	typeNumber  = "number"
	typeObject  = "object"
	typeString  = "string"
)

type openAPIDoc map[string]any

func generateOpenAPI(doc schemaDoc) ([]byte, error) {
	api, err := buildOpenAPIDoc(doc)
	if err != nil {
		return nil, err
	}
	return encodeOpenAPIYAML(api)
}

func buildOpenAPIDoc(doc schemaDoc) (openAPIDoc, error) {
	schemas, err := buildOpenAPISchemas(doc)
	if err != nil {
		return nil, err
	}

	return openAPIDoc{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "ColonyCore Entity Model",
			"version": doc.Version,
		},
		"components": map[string]any{
			"schemas": schemas,
		},
	}, nil
}

func buildOpenAPISchemas(doc schemaDoc) (map[string]any, error) {
	schemas := make(map[string]any, len(doc.Enums)+len(doc.Definitions)+len(doc.Entities)*3)

	for name, enum := range doc.Enums {
		schemas[toCamel(name)] = map[string]any{
			"type": typeString,
			"enum": toAnySlice(enum.Values),
		}
	}

	for name, def := range doc.Definitions {
		schema, err := schemaFromDefinition(def, doc.Enums, doc.Definitions)
		if err != nil {
			return nil, fmt.Errorf("definition %q: %w", name, err)
		}
		schemas[toCamel(name)] = schema
	}

	for name, ent := range doc.Entities {
		read, create, update, err := schemasFromEntity(ent, doc.Enums, doc.Definitions)
		if err != nil {
			return nil, fmt.Errorf("entity %q: %w", name, err)
		}
		schemas[name] = read
		schemas[name+"Create"] = create
		schemas[name+"Update"] = update
	}

	return schemas, nil
}

func schemaFromDefinition(def definitionSpec, enums map[string]enumSpec, defs map[string]definitionSpec) (map[string]any, error) {
	if len(def.Properties) == 0 && def.Ref == "" && def.Type != "" {
		return primitiveSchema(def), nil
	}

	return schemaForObject(def.Properties, def.Required, enums, defs, def.AdditionalProperties)
}

func schemasFromEntity(ent entitySpec, enums map[string]enumSpec, defs map[string]definitionSpec) (map[string]any, map[string]any, map[string]any, error) {
	props, required, err := propertiesForObject(ent.Properties, ent.Required, enums, defs)
	if err != nil {
		return nil, nil, nil, err
	}

	readProps := cloneMap(props)
	markReadOnly(readProps, []string{"id", "created_at", "updated_at"})

	read := map[string]any{
		"type":       "object",
		"properties": readProps,
	}
	if len(required) > 0 {
		read["required"] = required
	}

	createProps := cloneMap(readProps)
	for _, field := range []string{"id", "created_at", "updated_at"} {
		delete(createProps, field)
	}
	createRequired := filterRequired(required, []string{"id", "created_at", "updated_at"})
	create := map[string]any{
		"type":       "object",
		"properties": createProps,
	}
	if len(createRequired) > 0 {
		create["required"] = createRequired
	}

	updateProps := cloneMap(createProps)
	update := map[string]any{
		"type":       "object",
		"properties": updateProps,
	}

	return read, create, update, nil
}

func propertiesForObject(raw map[string]json.RawMessage, required []string, enums map[string]enumSpec, defs map[string]definitionSpec) (map[string]any, []string, error) {
	props := make(map[string]any, len(raw))
	parsed, _ := parseProperties(raw)
	for _, name := range sortedKeys(parsed) {
		prop := parsed[name]
		schema, err := schemaForProperty(prop, enums, defs)
		if err != nil {
			return nil, nil, fmt.Errorf("property %q: %w", name, err)
		}
		props[name] = schema
	}
	return props, cloneStrings(required), nil
}

func schemaForProperty(prop definitionSpec, enums map[string]enumSpec, defs map[string]definitionSpec) (map[string]any, error) {
	if prop.Ref != "" {
		ref := refToComponent(prop.Ref, enums, defs)
		if ref == "" {
			return nil, fmt.Errorf("unsupported ref %q", prop.Ref)
		}
		return map[string]any{"$ref": ref}, nil
	}

	switch prop.Type {
	case typeString, typeInteger, typeNumber, typeBoolean:
		return primitiveSchema(prop), nil
	case typeArray:
		items := map[string]any{}
		if prop.Items != nil {
			itemSchema, err := schemaForProperty(*prop.Items, enums, defs)
			if err != nil {
				return nil, err
			}
			items = itemSchema
		}
		return map[string]any{
			"type":  typeArray,
			"items": items,
		}, nil
	case typeObject:
		return schemaForObject(prop.Properties, prop.Required, enums, defs, prop.AdditionalProperties)
	default:
		return map[string]any{}, nil
	}
}

func schemaForObject(rawProps map[string]json.RawMessage, required []string, enums map[string]enumSpec, defs map[string]definitionSpec, additionalProps json.RawMessage) (map[string]any, error) {
	if len(rawProps) == 0 {
		schema := map[string]any{"type": typeObject}
		if val, ok := additionalPropertiesValue(additionalProps); ok {
			schema["additionalProperties"] = val
		}
		return schema, nil
	}

	props := make(map[string]any, len(rawProps))
	parsed, _ := parseProperties(rawProps)
	for _, name := range sortedKeys(parsed) {
		prop := parsed[name]
		schema, err := schemaForProperty(prop, enums, defs)
		if err != nil {
			return nil, err
		}
		props[name] = schema
	}

	result := map[string]any{
		"type":       typeObject,
		"properties": props,
	}
	if len(required) > 0 {
		result["required"] = cloneStrings(required)
	}
	if val, ok := additionalPropertiesValue(additionalProps); ok {
		result["additionalProperties"] = val
	}
	return result, nil
}

func primitiveSchema(prop definitionSpec) map[string]any {
	schema := map[string]any{
		"type": prop.Type,
	}
	if prop.Format != "" {
		schema["format"] = prop.Format
	}
	return schema
}

func refToComponent(ref string, enums map[string]enumSpec, defs map[string]definitionSpec) string {
	switch {
	case strings.HasPrefix(ref, "#/definitions/"):
		name := strings.TrimPrefix(ref, "#/definitions/")
		if defs != nil {
			if _, ok := defs[name]; ok {
				return "#/components/schemas/" + toCamel(name)
			}
		}
	case strings.HasPrefix(ref, "#/enums/"):
		name := strings.TrimPrefix(ref, "#/enums/")
		if enums != nil {
			if _, ok := enums[name]; ok {
				return "#/components/schemas/" + toCamel(name)
			}
		}
	}
	return ""
}

func markReadOnly(props map[string]any, names []string) {
	for _, name := range names {
		if schema, ok := props[name]; ok {
			if m, ok := schema.(map[string]any); ok {
				m["readOnly"] = true
			}
		}
	}
}

func filterRequired(required []string, remove []string) []string {
	filter := make([]string, 0, len(required))
	excluded := make(map[string]struct{}, len(remove))
	for _, name := range remove {
		excluded[strings.ToLower(name)] = struct{}{}
	}
	for _, name := range required {
		if _, ok := excluded[strings.ToLower(name)]; ok {
			continue
		}
		filter = append(filter, name)
	}
	sort.Strings(filter)
	return filter
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = cloneValue(v)
	}
	return dst
}

func cloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return cloneMap(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return val
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func toAnySlice(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func additionalPropertiesValue(raw json.RawMessage) (bool, bool) {
	if len(raw) == 0 {
		return false, false
	}
	var val bool
	if err := json.Unmarshal(raw, &val); err != nil {
		return false, false
	}
	return val, true
}

func encodeYAML(value any) ([]byte, error) {
	var b strings.Builder
	if err := writeYAML(&b, value, 0); err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	return []byte(b.String()), nil
}

func encodeOpenAPIYAML(doc openAPIDoc) ([]byte, error) {
	var b strings.Builder
	b.WriteString("# Code generated by internal/tools/entitymodel/generate. DO NOT EDIT.\n")
	b.WriteString("# Source of truth: docs/schema/entity-model.json\n")
	if err := writeYAML(&b, doc, 0); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func writeYAML(b *strings.Builder, value any, indent int) error {
	switch v := value.(type) {
	case openAPIDoc:
		return writeMapYAML(b, map[string]any(v), indent)
	case map[string]any:
		return writeMapYAML(b, v, indent)
	case []any:
		return writeSliceYAML(b, v, indent)
	case []string:
		items := toAnySlice(v)
		return writeSliceYAML(b, items, indent)
	default:
		writeScalarYAML(b, value, indent)
		return nil
	}
}

func writeMapYAML(b *strings.Builder, m map[string]any, indent int) error {
	if len(m) == 0 {
		writeIndented(b, "{}", indent)
		b.WriteByte('\n')
		return nil
	}
	for _, key := range sortedKeys(m) {
		writeIndented(b, key+":", indent)
		val := m[key]
		switch typed := val.(type) {
		case map[string]any:
			if len(typed) == 0 {
				b.WriteString(" {}\n")
				continue
			}
			b.WriteByte('\n')
			if err := writeMapYAML(b, typed, indent+1); err != nil {
				return err
			}
		case []any:
			if len(typed) == 0 {
				b.WriteString(" []\n")
				continue
			}
			b.WriteByte('\n')
			if err := writeSliceYAML(b, typed, indent+1); err != nil {
				return err
			}
		case []string:
			if len(typed) == 0 {
				b.WriteString(" []\n")
				continue
			}
			b.WriteByte('\n')
			if err := writeSliceYAML(b, toAnySlice(typed), indent+1); err != nil {
				return err
			}
		default:
			b.WriteByte(' ')
			writeScalarYAML(b, val, 0)
			b.WriteByte('\n')
		}
	}
	return nil
}

func writeSliceYAML(b *strings.Builder, list []any, indent int) error {
	for _, item := range list {
		writeIndented(b, "-", indent)
		switch val := item.(type) {
		case map[string]any:
			if len(val) == 0 {
				b.WriteString(" {}\n")
				continue
			}
			b.WriteByte(' ')
			if err := writeInlineOrNestedMap(b, val, indent); err != nil {
				return err
			}
		case []any:
			if len(val) == 0 {
				b.WriteString(" []\n")
				continue
			}
			b.WriteByte('\n')
			if err := writeSliceYAML(b, val, indent+1); err != nil {
				return err
			}
		case []string:
			if len(val) == 0 {
				b.WriteString(" []\n")
				continue
			}
			b.WriteByte('\n')
			if err := writeSliceYAML(b, toAnySlice(val), indent+1); err != nil {
				return err
			}
		default:
			b.WriteByte(' ')
			writeScalarYAML(b, val, 0)
			b.WriteByte('\n')
		}
	}
	return nil
}

func writeInlineOrNestedMap(b *strings.Builder, val map[string]any, indent int) error {
	if len(val) == 0 {
		b.WriteString("{}\n")
		return nil
	}

	b.WriteByte('\n')
	return writeMapYAML(b, val, indent+1)
}

func writeScalarYAML(b *strings.Builder, value any, indent int) {
	writeIndented(b, formatScalar(value), indent)
}

func writeIndented(b *strings.Builder, value string, indent int) {
	if indent > 0 {
		b.WriteString(strings.Repeat(" ", indent*2))
	}
	b.WriteString(value)
}

func formatScalar(value any) string {
	switch v := value.(type) {
	case string:
		return strconv.Quote(v)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return "null"
	case int, int64, float64, float32:
		return fmt.Sprint(v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}
