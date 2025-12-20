package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJoinTableGeneratedForArrayRelationship(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"SupplyItem": {
				Required: []string{"id", "created_at", "updated_at", "name"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":       raw(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{},
			},
			"Project": {
				Required: []string{"id", "created_at", "updated_at", "name", "supply_item_ids"},
				Properties: map[string]json.RawMessage{
					"id":              raw(`{"$ref":"#/definitions/id"}`),
					"created_at":      raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at":      raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":            raw(`{"type":"string"}`),
					"supply_item_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"supply_item_ids": {Target: "SupplyItem", Cardinality: "0..n"},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}

	if strings.Contains(sql, "supply_item_ids JSONB") {
		t.Fatalf("expected array relationship to be projected as join table, got column: %s", sql)
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS projects__supply_item_ids") {
		t.Fatalf("expected join table for array relationship:\n%s", sql)
	}
	if !strings.Contains(sql, "project_id UUID NOT NULL") || !strings.Contains(sql, "supply_item_id UUID NOT NULL") {
		t.Fatalf("expected join table columns for source/target IDs:\n%s", sql)
	}
	if !strings.Contains(sql, "FOREIGN KEY (supply_item_id) REFERENCES supply_items(id)") {
		t.Fatalf("expected FK to target in join table:\n%s", sql)
	}
}

func TestDerivedArrayWhenInverseFK(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"Parent": {
				Required: []string{"id", "created_at", "updated_at"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"child_ids":  raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"child_ids": {Target: "Child", Cardinality: "0..n"},
				},
			},
			"Child": {
				Required: []string{"id", "created_at", "updated_at", "parent_id"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"parent_id":  raw(`{"$ref":"#/definitions/entity_id"}`),
				},
				Relationships: map[string]relationshipSpec{
					"parent_id": {Target: "Parent", Cardinality: "1..1"},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}

	if strings.Contains(sql, "child_ids JSONB") || strings.Contains(sql, "parents__child_ids") {
		t.Fatalf("expected derived array to be omitted (no column/join table):\n%s", sql)
	}
	if !strings.Contains(sql, "parent_id UUID NOT NULL") {
		t.Fatalf("expected inverse FK column to be required:\n%s", sql)
	}
}

func TestSelfJoinUsesRelationshipColumnForTarget(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"Organism": {
				Required: []string{"id", "created_at", "updated_at", "parent_ids"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"parent_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"parent_ids": {Target: "Organism", Cardinality: "0..n"},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}
	if !strings.Contains(sql, "organisms__parent_ids") {
		t.Fatalf("expected self-join table:\n%s", sql)
	}
	if !strings.Contains(sql, "organism_id UUID NOT NULL") || !strings.Contains(sql, "parent_ids_id UUID NOT NULL") {
		t.Fatalf("expected distinct source/target columns in self-join:\n%s", sql)
	}
	if strings.Contains(sql, "PRIMARY KEY (organism_id, organism_id)") {
		t.Fatalf("expected no duplicate PK columns:\n%s", sql)
	}
}

func TestJSONStorageKeepsArrayColumn(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
		},
		Entities: map[string]entitySpec{
			"Artifact": {
				Required: []string{"id", "tags"},
				Properties: map[string]json.RawMessage{
					"id":   raw(`{"$ref":"#/definitions/id"}`),
					"tags": raw(`{"type":"array","items":{"type":"string"}}`),
				},
				Relationships: map[string]relationshipSpec{
					"tags": {Target: "Artifact", Cardinality: "0..n", Storage: storageJSON},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}
	if !strings.Contains(sql, "tags JSONB") {
		t.Fatalf("expected JSON storage to keep array column:\n%s", sql)
	}
}

func TestInvalidStorageCombination(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
		},
		Entities: map[string]entitySpec{
			"Thing": {
				Required: []string{"id", "items"},
				Properties: map[string]json.RawMessage{
					"id":    raw(`{"$ref":"#/definitions/id"}`),
					"items": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"}}`),
				},
				Relationships: map[string]relationshipSpec{
					"items": {Target: "Thing", Cardinality: "0..n", Storage: storageFK},
				},
			},
		},
	}

	if _, err := buildSQLForDialect(doc, postgresDialect); err == nil || !strings.Contains(err.Error(), "uses fk storage but property is an array") {
		t.Fatalf("expected FK-on-array error, got: %v", err)
	}
}

func TestIsArrayPropertyCoversRefResolutionErrors(t *testing.T) {
	enums := map[string]enumSpec{
		"status": {Values: []string{"ok"}},
	}
	defs := map[string]definitionSpec{
		"id": {Type: typeString},
	}
	if !isArrayProperty(definitionSpec{Type: typeArray, Items: &definitionSpec{Type: typeString}}, enums, defs) {
		t.Fatalf("expected direct array type to be detected")
	}
	if isArrayProperty(definitionSpec{Ref: "#/definitions/missing"}, enums, defs) {
		t.Fatalf("expected unresolved ref to return false")
	}
}

func TestASCIILowerHelpers(t *testing.T) {
	if toLowerASCII('A') != 'a' || toLowerASCII('z') != 'z' {
		t.Fatalf("toLowerASCII did not lower case as expected")
	}
	if !isUpperASCII('Z') || isUpperASCII('a') {
		t.Fatalf("isUpperASCII returned incorrect result")
	}
	if !isLowerASCII('z') || isLowerASCII('A') {
		t.Fatalf("isLowerASCII returned incorrect result")
	}
}

func TestJoinTablesDeduplicated(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
		},
		Entities: map[string]entitySpec{
			"Facility": {
				Required: []string{"id", "name"},
				Properties: map[string]json.RawMessage{
					"id":   raw(`{"$ref":"#/definitions/id"}`),
					"name": raw(`{"type":"string"}`),
					"project_ids": raw(`{
						"type":"array",
						"items":{"$ref":"#/definitions/entity_id"},
						"uniqueItems":true
					}`),
				},
				Relationships: map[string]relationshipSpec{
					"project_ids": {Target: "Project", Cardinality: "0..n"},
				},
			},
			"Project": {
				Required: []string{"id", "name", "facility_ids"},
				Properties: map[string]json.RawMessage{
					"id":           raw(`{"$ref":"#/definitions/id"}`),
					"name":         raw(`{"type":"string"}`),
					"facility_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"facility_ids": {Target: "Facility", Cardinality: "0..n"},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}

	if strings.Count(sql, "CREATE TABLE IF NOT EXISTS facilities__project_ids") != 1 {
		t.Fatalf("expected single join table for facilities<->projects:\n%s", sql)
	}
	if strings.Contains(sql, "CREATE TABLE IF NOT EXISTS projects__facility_ids") {
		t.Fatalf("expected deduped join table name, found projects__facility_ids:\n%s", sql)
	}
	if strings.Contains(sql, "project_ids JSONB") || strings.Contains(sql, "facility_ids JSONB") {
		t.Fatalf("expected relationship arrays to be projected to join tables only:\n%s", sql)
	}
}

func TestParallelArrayRelationshipsKeepDistinctJoins(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"Organism": {
				Required: []string{"id", "created_at", "updated_at", "name"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":       raw(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{},
			},
			"BreedingUnit": {
				Required: []string{"id", "created_at", "updated_at", "name"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":       raw(`{"type":"string"}`),
					"female_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
					"male_ids":   raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"female_ids": {Target: "Organism", Cardinality: "0..n"},
					"male_ids":   {Target: "Organism", Cardinality: "0..n"},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}

	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS breeding_units__female_ids") {
		t.Fatalf("expected female join table to be present:\n%s", sql)
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS breeding_units__male_ids") {
		t.Fatalf("expected male join table to be present:\n%s", sql)
	}
}

func TestRequiredJoinTriggersUniquePerJoinTable(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
			"timestamp": {Type: typeString, Format: dateTimeFormat},
		},
		Entities: map[string]entitySpec{
			"Facility": {
				Required: []string{"id", "created_at", "updated_at", "name"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":       raw(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{},
			},
			"Protocol": {
				Required: []string{"id", "created_at", "updated_at", "title"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"title":      raw(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{},
			},
			"Project": {
				Required: []string{"id", "created_at", "updated_at", "title"},
				Properties: map[string]json.RawMessage{
					"id":         raw(`{"$ref":"#/definitions/id"}`),
					"created_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at": raw(`{"$ref":"#/definitions/timestamp"}`),
					"title":      raw(`{"type":"string"}`),
				},
				Relationships: map[string]relationshipSpec{},
			},
			"Permit": {
				Required: []string{"id", "created_at", "updated_at", "facility_ids", "protocol_ids"},
				Properties: map[string]json.RawMessage{
					"id":           raw(`{"$ref":"#/definitions/id"}`),
					"created_at":   raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at":   raw(`{"$ref":"#/definitions/timestamp"}`),
					"facility_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
					"protocol_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"facility_ids": {Target: "Facility", Cardinality: "0..n"},
					"protocol_ids": {Target: "Protocol", Cardinality: "0..n"},
				},
			},
			"SupplyItem": {
				Required: []string{"id", "created_at", "updated_at", "name", "facility_ids", "project_ids"},
				Properties: map[string]json.RawMessage{
					"id":           raw(`{"$ref":"#/definitions/id"}`),
					"created_at":   raw(`{"$ref":"#/definitions/timestamp"}`),
					"updated_at":   raw(`{"$ref":"#/definitions/timestamp"}`),
					"name":         raw(`{"type":"string"}`),
					"facility_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
					"project_ids":  raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"facility_ids": {Target: "Facility", Cardinality: "0..n"},
					"project_ids":  {Target: "Project", Cardinality: "0..n"},
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}

	if !strings.Contains(sql, "permits_permits__facility_ids_permit_id_required") {
		t.Fatalf("expected required trigger for permit facilities:\n%s", sql)
	}
	if !strings.Contains(sql, "permits_permits__protocol_ids_permit_id_required") {
		t.Fatalf("expected required trigger for permit protocols:\n%s", sql)
	}
	if !strings.Contains(sql, "supply_items_supply_items__facility_ids_supply_item_id_required") {
		t.Fatalf("expected required trigger for supply item facilities:\n%s", sql)
	}
	if !strings.Contains(sql, "supply_items_supply_items__project_ids_supply_item_id_required") && !strings.Contains(sql, "supply_items_projects__supply_item_ids_supply_item_id_required") {
		t.Fatalf("expected required trigger for supply item projects:\n%s", sql)
	}
}

func TestEnumChecksAndNaturalKeys(t *testing.T) {
	doc := schemaDoc{
		Enums: map[string]enumSpec{
			"status": {Values: []string{"active", "archived"}},
		},
		Definitions: map[string]definitionSpec{
			"id": {Type: typeString, Format: "uuid"},
		},
		Entities: map[string]entitySpec{
			"Thing": {
				Required: []string{"id", "code", "status"},
				NaturalKeys: []naturalKeySpec{
					{Fields: []string{"code"}, Scope: "global"},
				},
				Properties: map[string]json.RawMessage{
					"id":     raw(`{"$ref":"#/definitions/id"}`),
					"code":   raw(`{"type":"string"}`),
					"status": raw(`{"$ref":"#/enums/status"}`),
					"state":  raw(`{"$ref":"#/enums/status"}`),
				},
			},
		},
	}

	sql, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect: %v", err)
	}

	if !strings.Contains(sql, "CHECK (status IN ('active', 'archived'))") {
		t.Fatalf("expected enum check constraint for required field:\n%s", sql)
	}
	if !strings.Contains(sql, "CHECK ((state IN ('active', 'archived') OR state IS NULL))") {
		t.Fatalf("expected nullable enum check constraint for optional field:\n%s", sql)
	}
	if !strings.Contains(sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_things_nk_1 ON things (code)") {
		t.Fatalf("expected unique index for natural key:\n%s", sql)
	}
}

func TestRequiredJoinEnforcementTriggers(t *testing.T) {
	doc := schemaDoc{
		Definitions: map[string]definitionSpec{
			"id":        {Type: typeString, Format: "uuid"},
			"entity_id": {Type: typeString, Format: "uuid"},
		},
		Entities: map[string]entitySpec{
			"Facility": {
				Required: []string{"id", "name"},
				Properties: map[string]json.RawMessage{
					"id":   raw(`{"$ref":"#/definitions/id"}`),
					"name": raw(`{"type":"string"}`),
				},
			},
			"Project": {
				Required: []string{"id", "name", "facility_ids"},
				Properties: map[string]json.RawMessage{
					"id":           raw(`{"$ref":"#/definitions/id"}`),
					"name":         raw(`{"type":"string"}`),
					"facility_ids": raw(`{"type":"array","items":{"$ref":"#/definitions/entity_id"},"uniqueItems":true}`),
				},
				Relationships: map[string]relationshipSpec{
					"facility_ids": {Target: "Facility", Cardinality: "0..n"},
				},
			},
		},
	}

	pgSQL, err := buildSQLForDialect(doc, postgresDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect postgres: %v", err)
	}
	if !strings.Contains(pgSQL, "enforce_required_join_parent") || !strings.Contains(pgSQL, "enforce_required_join_link") {
		t.Fatalf("expected required-join enforcement helpers in postgres SQL:\n%s", pgSQL)
	}
	if !strings.Contains(pgSQL, "CREATE CONSTRAINT TRIGGER projects_projects__facility_ids_project_id_required") {
		t.Fatalf("expected parent constraint trigger for required join:\n%s", pgSQL)
	}
	if !strings.Contains(pgSQL, "CREATE CONSTRAINT TRIGGER projects__facility_ids_project_id_guard") {
		t.Fatalf("expected join-table guard trigger for required join:\n%s", pgSQL)
	}

	sqliteSQL, err := buildSQLForDialect(doc, sqliteDialect)
	if err != nil {
		t.Fatalf("buildSQLForDialect sqlite: %v", err)
	}
	if strings.Contains(sqliteSQL, "enforce_required_join_parent") || strings.Contains(sqliteSQL, "enforce_required_join_link") {
		t.Fatalf("did not expect required-join enforcement helpers in sqlite output:\n%s", sqliteSQL)
	}
}
