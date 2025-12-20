package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const timestampDefinition = "timestamp"

type contractMetadata struct {
	Version  string                            `json:"version"`
	Entities map[string]contractEntityMetadata `json:"entities"`
}

type contractEntityMetadata struct {
	Required       []string `json:"required"`
	ExtensionHooks []string `json:"extension_hooks"`
}

func generatePluginContract(doc schemaDoc) ([]byte, error) {
	meta := buildContractMetadata(doc)

	var b strings.Builder
	b.WriteString("# Plugin Contract (Entity Model v0)\n\n")
	status := strings.TrimSpace(doc.Metadata.Status)
	if status == "" {
		status = "unknown"
	}
	fmt.Fprintf(&b, "_Source: `docs/schema/entity-model.json` v%s (status: %s)._\n\n", strings.TrimSpace(doc.Version), status)
	b.WriteString("This document enumerates the canonical fields, relationships, extension hooks, and invariants each plugin must respect. Generate it via `make entity-model-generate`.\n\n")

	writeIDSemantics(&b, doc.IDSemantics)
	writeEnumsSection(&b, doc.Enums)
	writeEntitiesSection(&b, doc)

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal contract metadata: %w", err)
	}
	b.WriteString("<!--\nCONTRACT-METADATA\n")
	b.Write(metaBytes)
	b.WriteString("\n-->\n")

	return []byte(b.String()), nil
}

func writeIDSemantics(b *strings.Builder, idSpec *idSemanticsSpec) {
	b.WriteString("## ID Semantics\n\n")
	if idSpec == nil {
		b.WriteString("_Not declared in schema._\n\n")
		return
	}
	fmt.Fprintf(b, "- Type: `%s`\n", strings.TrimSpace(idSpec.Type))
	fmt.Fprintf(b, "- Scope: %s\n", fallbackText(idSpec.Scope))
	fmt.Fprintf(b, "- Required: %t\n", idSpec.Required)
	if desc := strings.TrimSpace(idSpec.Description); desc != "" {
		fmt.Fprintf(b, "- Description: %s\n", desc)
	}
	b.WriteString("\n")
}

func writeEnumsSection(b *strings.Builder, enums map[string]enumSpec) {
	b.WriteString("## Enums\n\n")
	if len(enums) == 0 {
		b.WriteString("_None declared._\n\n")
		return
	}
	b.WriteString("| Name | Values | Initial | Terminal | Description |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, name := range sortedKeys(enums) {
		enum := enums[name]
		values := joinInline(enum.Values)
		initial := dashIfEmpty(enum.Initial)
		if initial != "-" {
			initial = fmt.Sprintf("`%s`", initial)
		}
		terminal := joinInline(enum.Terminal)
		if terminal == "-" && len(enum.Terminal) == 0 {
			terminal = "-"
		}
		desc := fallbackText(enum.Description)
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", toCamel(name), values, initial, terminal, desc)
	}
	b.WriteString("\n")
}

func writeEntitiesSection(b *strings.Builder, doc schemaDoc) {
	b.WriteString("## Entities\n\n")
	names := sortedKeys(doc.Entities)
	if len(names) == 0 {
		b.WriteString("_No entities defined._\n\n")
		return
	}
	for _, name := range names {
		ent := doc.Entities[name]
		fmt.Fprintf(b, "### %s\n\n", name)
		if desc := strings.TrimSpace(ent.Description); desc != "" {
			b.WriteString(desc + "\n\n")
		}

		fmt.Fprintf(b, "**Required fields:** %s\n\n", inlineCodeList(ent.Required))
		writeNaturalKeys(b, ent.NaturalKeys)
		writeStateBlock(b, ent.States)
		writeInvariants(b, ent.Invariants)
		writeRelationshipsTable(b, ent)
		writeExtensionHooks(b, ent)
		writeFieldsTable(b, ent, doc.Enums, doc.Definitions)
	}
}

func writeNaturalKeys(b *strings.Builder, keys []naturalKeySpec) {
	b.WriteString("**Natural keys:**\n\n")
	if len(keys) == 0 {
		b.WriteString("_none_\n\n")
		return
	}
	for _, key := range keys {
		fields := inlineCodeList(key.Fields)
		scope := fallbackText(key.Scope)
		if desc := strings.TrimSpace(key.Description); desc != "" {
			fmt.Fprintf(b, "- %s (scope: %s) — %s\n", fields, scope, desc)
		} else {
			fmt.Fprintf(b, "- %s (scope: %s)\n", fields, scope)
		}
	}
	b.WriteString("\n")
}

func writeStateBlock(b *strings.Builder, state *stateSpec) {
	b.WriteString("**States:** ")
	if state == nil || strings.TrimSpace(state.Enum) == "" {
		b.WriteString("_none declared._\n\n")
		return
	}
	initial := dashIfEmpty(state.Initial)
	if initial != "-" {
		initial = fmt.Sprintf("`%s`", initial)
	}
	terminal := inlineCodeList(state.Terminal)
	fmt.Fprintf(b, "Enum `%s` (initial %s; terminal: %s).\n\n", toCamel(state.Enum), initial, terminal)
}

func writeInvariants(b *strings.Builder, invariants []string) {
	b.WriteString("**Invariants:** ")
	if len(invariants) == 0 {
		b.WriteString("_none declared._\n\n")
		return
	}
	b.WriteString(inlineCodeList(invariants) + "\n\n")
}

func writeRelationshipsTable(b *strings.Builder, ent entitySpec) {
	b.WriteString("**Relationships**\n\n")
	if len(ent.Relationships) == 0 {
		b.WriteString("_none_\n\n")
		return
	}
	b.WriteString("| Field | Target | Cardinality | Storage |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, relName := range sortedKeys(ent.Relationships) {
		rel := ent.Relationships[relName]
		storage := strings.TrimSpace(rel.Storage)
		if storage == "" {
			storage = "fk"
		}
		fmt.Fprintf(b, "| `%s` | %s | %s | %s |\n", relName, rel.Target, rel.Cardinality, storage)
	}
	b.WriteString("\n")
}

func writeExtensionHooks(b *strings.Builder, ent entitySpec) {
	hooks := extensionHooksForEntity(ent)
	b.WriteString("**Extension hooks:** ")
	if len(hooks) == 0 {
		b.WriteString("_none_.\n\n")
		return
	}
	b.WriteString(inlineCodeList(hooks) + "\n\n")
}

func writeFieldsTable(b *strings.Builder, ent entitySpec, enums map[string]enumSpec, defs map[string]definitionSpec) {
	b.WriteString("**Fields**\n\n")
	props, _ := parseProperties(ent.Properties)
	if len(props) == 0 {
		b.WriteString("_no fields defined._\n\n")
		return
	}
	b.WriteString("| Field | Type | Required | Notes |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	requiredSet := make(map[string]struct{}, len(ent.Required))
	for _, name := range ent.Required {
		requiredSet[name] = struct{}{}
	}
	for _, propName := range sortedKeys(props) {
		prop := props[propName]
		_, isRequired := requiredSet[propName]
		typeDesc := propertyTypeString(prop, enums, defs)
		note := fallbackText(prop.Description)
		reqText := "No"
		if isRequired {
			reqText = "Yes"
		}
		fmt.Fprintf(b, "| `%s` | `%s` | %s | %s |\n", propName, typeDesc, reqText, note)
	}
	b.WriteString("\n")
}

func propertyTypeString(prop definitionSpec, enums map[string]enumSpec, defs map[string]definitionSpec) string {
	if prop.Ref != "" {
		switch {
		case strings.HasPrefix(prop.Ref, "#/definitions/"):
			name := strings.TrimPrefix(prop.Ref, "#/definitions/")
			if name == timestampDefinition {
				return timestampDefinition
			}
			if def, ok := defs[name]; ok {
				if def.Format != "" {
					return def.Format
				}
			}
			return toCamel(name)
		case strings.HasPrefix(prop.Ref, "#/enums/"):
			name := strings.TrimPrefix(prop.Ref, "#/enums/")
			return "enum " + toCamel(name)
		}
	}

	switch prop.Type {
	case typeString:
		if prop.Format != "" {
			return prop.Format
		}
		return typeString
	case typeInteger:
		return "integer"
	case typeNumber:
		return "number"
	case typeBoolean:
		return "boolean"
	case typeArray:
		if prop.Items == nil {
			return "array<any>"
		}
		return fmt.Sprintf("array<%s>", propertyTypeString(*prop.Items, enums, defs))
	case typeObject:
		if allowsAdditionalProperties(prop.AdditionalProperties) {
			return "map<string, any>"
		}
		return "object"
	default:
		if prop.Items != nil {
			return fmt.Sprintf("array<%s>", propertyTypeString(*prop.Items, enums, defs))
		}
	}
	return "any"
}

func extensionHooksForEntity(ent entitySpec) []string {
	props, _ := parseProperties(ent.Properties)
	hooks := make([]string, 0)
	for name, prop := range props {
		if prop.Ref == "#/definitions/extension_attributes" {
			hooks = append(hooks, name)
		}
	}
	sort.Strings(hooks)
	return hooks
}

func buildContractMetadata(doc schemaDoc) contractMetadata {
	entities := make(map[string]contractEntityMetadata, len(doc.Entities))
	for _, name := range sortedKeys(doc.Entities) {
		ent := doc.Entities[name]
		req := append([]string(nil), ent.Required...)
		sort.Strings(req)
		hooks := extensionHooksForEntity(ent)
		entities[name] = contractEntityMetadata{Required: req, ExtensionHooks: hooks}
	}
	return contractMetadata{Version: strings.TrimSpace(doc.Version), Entities: entities}
}

func inlineCodeList(values []string) string {
	if len(values) == 0 {
		return "_none_"
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("`%s`", v)
	}
	return strings.Join(quoted, ", ")
}

func joinInline(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("`%s`", v)
	}
	return strings.Join(quoted, "<br>")
}

func dashIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func fallbackText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
