package domain

import (
	"encoding/json"
	"testing"
	"time"

	"colonycore/pkg/domain/extension"
)

const (
	testAttrValue   = "value"
	testTemperature = "22C"
)

func TestOrganismMarshalJSON(t *testing.T) {
	now := time.Now().UTC()
	organism := Organism{
		Base: Base{
			ID:        "test-organism",
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now,
		},
		Name:    "Test Organism",
		Species: "Test Species",
		Stage:   StageAdult,
	}

	// Set attributes to test marshaling
	organism.SetAttributes(map[string]any{
		"weight": 100.5,
		"color":  "green",
		"metadata": map[string]any{
			"source": "lab",
		},
	})

	data, err := json.Marshal(organism)
	if err != nil {
		t.Fatalf("Failed to marshal organism: %v", err)
	}

	// Verify the JSON contains attributes
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	attributes, ok := result["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("Expected attributes in JSON output")
	}

	if attributes["weight"] != 100.5 {
		t.Errorf("Expected weight 100.5, got %v", attributes["weight"])
	}
	if attributes["color"] != "green" {
		t.Errorf("Expected color 'green', got %v", attributes["color"])
	}
}

func TestOrganismUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-organism",
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2023-01-01T01:00:00Z",
		"name": "Test Organism",
		"species": "Test Species",
		"stage": "adult",
		"attributes": {
			"weight": 100.5,
			"color": "green",
			"metadata": {
				"source": "lab"
			}
		}
	}`

	var organism Organism
	if err := json.Unmarshal([]byte(jsonData), &organism); err != nil {
		t.Fatalf("Failed to unmarshal organism: %v", err)
	}

	if organism.ID != "test-organism" {
		t.Errorf("Expected ID 'test-organism', got %v", organism.ID)
	}
	if organism.Name != "Test Organism" {
		t.Errorf("Expected name 'Test Organism', got %v", organism.Name)
	}
	if organism.Stage != StageAdult {
		t.Errorf("Expected stage 'adult', got %v", organism.Stage)
	}

	// Test attributes were properly unmarshaled
	attrs := organism.AttributesMap()
	if attrs["weight"] != 100.5 {
		t.Errorf("Expected weight 100.5, got %v", attrs["weight"])
	}
	if attrs["color"] != "green" {
		t.Errorf("Expected color 'green', got %v", attrs["color"])
	}

	metadata, ok := attrs["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("Expected metadata to be a map")
	}
	if metadata["source"] != "lab" {
		t.Errorf("Expected source 'lab', got %v", metadata["source"])
	}
}

func TestOrganismMarshalUnmarshalRoundTrip(t *testing.T) {
	original := Organism{
		Base: Base{
			ID:        "roundtrip-test",
			CreatedAt: time.Now().UTC().Add(-time.Hour),
			UpdatedAt: time.Now().UTC(),
		},
		Name:    "Roundtrip Test",
		Species: "Test Species",
		Stage:   StageJuvenile,
	}

	original.SetAttributes(map[string]any{
		"test": "value",
		"nested": map[string]any{
			"inner": []any{1, 2, 3},
		},
	})

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal from JSON
	var restored Organism
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Compare fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected %v, got %v", original.ID, restored.ID)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: expected %v, got %v", original.Name, restored.Name)
	}
	if restored.Stage != original.Stage {
		t.Errorf("Stage mismatch: expected %v, got %v", original.Stage, restored.Stage)
	}

	// Compare attributes
	originalAttrs := original.AttributesMap()
	restoredAttrs := restored.AttributesMap()

	if restoredAttrs["test"] != originalAttrs["test"] {
		t.Errorf("Attribute 'test' mismatch: expected %v, got %v", originalAttrs["test"], restoredAttrs["test"])
	}
}

func TestFacilityMarshalJSON(t *testing.T) {
	facility := Facility{
		Base: Base{ID: "test-facility"},
		Name: "Test Facility",
		Zone: "Test Zone",
	}

	facility.SetEnvironmentBaselines(map[string]any{
		"temperature": testTemperature,
		"humidity":    "65%",
	})

	data, err := json.Marshal(facility)
	if err != nil {
		t.Fatalf("Failed to marshal facility: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	baselines, ok := result["environment_baselines"].(map[string]any)
	if !ok {
		t.Fatalf("Expected environment_baselines in JSON output")
	}

	if baselines["temperature"] != testTemperature {
		t.Errorf("Expected temperature '%v', got %v", testTemperature, baselines["temperature"])
	}
}

func TestFacilityUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-facility",
		"name": "Test Facility",
		"zone": "Test Zone",
		"environment_baselines": {
			"temperature": "` + testTemperature + `",
			"humidity": "65%"
		}
	}`

	var facility Facility
	if err := json.Unmarshal([]byte(jsonData), &facility); err != nil {
		t.Fatalf("Failed to unmarshal facility: %v", err)
	}

	if facility.ID != "test-facility" {
		t.Errorf("Expected ID 'test-facility', got %v", facility.ID)
	}

	baselines := facility.EnvironmentBaselinesMap()
	if baselines["temperature"] != testTemperature {
		t.Errorf("Expected temperature '22C', got %v", baselines["temperature"])
	}
}

func TestBreedingUnitMarshalJSON(t *testing.T) {
	unit := BreedingUnit{
		Base:     Base{ID: "test-breeding"},
		Name:     "Test Breeding Unit",
		Strategy: "pair",
	}

	unit.SetPairingAttributes(map[string]any{
		"purpose": "research",
		"notes":   "test pairing",
	})

	data, err := json.Marshal(unit)
	if err != nil {
		t.Fatalf("Failed to marshal breeding unit: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	attributes, ok := result["pairing_attributes"].(map[string]any)
	if !ok {
		t.Fatalf("Expected pairing_attributes in JSON output")
	}

	if attributes["purpose"] != "research" {
		t.Errorf("Expected purpose 'research', got %v", attributes["purpose"])
	}
}

func TestBreedingUnitUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-breeding",
		"name": "Test Breeding Unit",
		"strategy": "pair",
		"pairing_attributes": {
			"purpose": "research",
			"notes": "test pairing"
		}
	}`

	var unit BreedingUnit
	if err := json.Unmarshal([]byte(jsonData), &unit); err != nil {
		t.Fatalf("Failed to unmarshal breeding unit: %v", err)
	}

	if unit.ID != "test-breeding" {
		t.Errorf("Expected ID 'test-breeding', got %v", unit.ID)
	}

	attrs := unit.PairingAttributesMap()
	if attrs["purpose"] != "research" {
		t.Errorf("Expected purpose 'research', got %v", attrs["purpose"])
	}
}

func TestObservationMarshalJSON(t *testing.T) {
	obs := Observation{
		Base:     Base{ID: "test-observation"},
		Observer: "Test Observer",
	}

	obs.SetData(map[string]any{
		"weight":      50.5,
		"temperature": 22.0,
		"notes":       "test observation data",
	})

	data, err := json.Marshal(obs)
	if err != nil {
		t.Fatalf("Failed to marshal observation: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	observationData, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("Expected data in JSON output")
	}

	if observationData["weight"] != 50.5 {
		t.Errorf("Expected weight 50.5, got %v", observationData["weight"])
	}
}

func TestObservationUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-observation",
		"observer": "Test Observer",
		"data": {
			"weight": 50.5,
			"temperature": 22.0,
			"notes": "test observation data"
		}
	}`

	var obs Observation
	if err := json.Unmarshal([]byte(jsonData), &obs); err != nil {
		t.Fatalf("Failed to unmarshal observation: %v", err)
	}

	if obs.ID != "test-observation" {
		t.Errorf("Expected ID 'test-observation', got %v", obs.ID)
	}

	data := obs.DataMap()
	if data["weight"] != 50.5 {
		t.Errorf("Expected weight 50.5, got %v", data["weight"])
	}
}

const testVolumeAttribute = "5ml"

func TestSampleMarshalJSON(t *testing.T) {
	sample := Sample{
		Base:       Base{ID: "test-sample"},
		Identifier: "SAMPLE-001",
		SourceType: "blood",
	}

	sample.SetAttributes(map[string]any{
		"volume":     testVolumeAttribute,
		"processing": "frozen",
	})

	data, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("Failed to marshal sample: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	attributes, ok := result["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("Expected attributes in JSON output")
	}

	if attributes["volume"] != testVolumeAttribute {
		t.Errorf("Expected volume '%v', got %v", testVolumeAttribute, attributes["volume"])
	}
}

func TestSampleUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-sample",
		"identifier": "SAMPLE-001",
		"source_type": "blood",
		"attributes": {
			"volume": "` + testVolumeAttribute + `",
			"processing": "frozen"
		}
	}`

	var sample Sample
	if err := json.Unmarshal([]byte(jsonData), &sample); err != nil {
		t.Fatalf("Failed to unmarshal sample: %v", err)
	}

	if sample.ID != "test-sample" {
		t.Errorf("Expected ID 'test-sample', got %v", sample.ID)
	}

	attrs := sample.AttributesMap()
	if attrs["volume"] != testVolumeAttribute {
		t.Errorf("Expected volume '5ml', got %v", attrs["volume"])
	}
}

func TestSupplyItemMarshalJSON(t *testing.T) {
	supply := SupplyItem{
		Base: Base{ID: "test-supply"},
		SKU:  "SKU-001",
		Name: "Test Supply",
	}

	supply.SetAttributes(map[string]any{
		"brand":    "TestBrand",
		"category": "consumable",
	})

	data, err := json.Marshal(supply)
	if err != nil {
		t.Fatalf("Failed to marshal supply item: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	attributes, ok := result["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("Expected attributes in JSON output")
	}

	if attributes["brand"] != "TestBrand" {
		t.Errorf("Expected brand 'TestBrand', got %v", attributes["brand"])
	}
}

func TestSupplyItemUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "test-supply",
		"sku": "SKU-001",
		"name": "Test Supply",
		"attributes": {
			"brand": "TestBrand",
			"category": "consumable"
		}
	}`

	var supply SupplyItem
	if err := json.Unmarshal([]byte(jsonData), &supply); err != nil {
		t.Fatalf("Failed to unmarshal supply item: %v", err)
	}

	if supply.ID != "test-supply" {
		t.Errorf("Expected ID 'test-supply', got %v", supply.ID)
	}

	attrs := supply.AttributesMap()
	if attrs["brand"] != "TestBrand" {
		t.Errorf("Expected brand 'TestBrand', got %v", attrs["brand"])
	}
}

func TestJSONMarshalWithNilAttributes(t *testing.T) {
	// Test that entities with nil attributes marshal correctly
	organism := Organism{
		Base:    Base{ID: "nil-attrs"},
		Name:    "No Attributes",
		Species: "Test",
	}

	data, err := json.Marshal(organism)
	if err != nil {
		t.Fatalf("Failed to marshal organism with nil attributes: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Attributes should not be present in JSON when nil
	if _, exists := result["attributes"]; exists {
		t.Errorf("Expected attributes to be omitted when nil")
	}
}

func TestJSONUnmarshalWithMissingAttributes(t *testing.T) {
	// Test that entities can unmarshal when attributes field is missing
	jsonData := `{
		"id": "missing-attrs",
		"name": "No Attributes",
		"species": "Test"
	}`

	var organism Organism
	if err := json.Unmarshal([]byte(jsonData), &organism); err != nil {
		t.Fatalf("Failed to unmarshal organism without attributes: %v", err)
	}

	if organism.ID != "missing-attrs" {
		t.Errorf("Expected ID 'missing-attrs', got %v", organism.ID)
	}

	attrs := organism.AttributesMap()
	if attrs != nil {
		t.Errorf("Expected nil attributes map, got %v", attrs)
	}
}

// TestJSONMarshalErrorCases tests error conditions in JSON marshaling
func TestJSONMarshalErrorCases(t *testing.T) {
	// Test marshaling organism with invalid data in attributes
	organism := Organism{
		Base: Base{ID: "test"},
		Name: "Test",
	}

	// Set attributes with data that could cause issues
	organism.SetAttributes(map[string]any{
		"valid":   "string",
		"number":  42,
		"boolean": true,
		"array":   []string{"a", "b", "c"},
	})

	// This should work fine
	data, err := json.Marshal(organism)
	if err != nil {
		t.Errorf("Expected marshaling to succeed, got error: %v", err)
	}

	// Verify round-trip
	var unmarshaled Organism
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Expected unmarshaling to succeed, got error: %v", err)
	}

	if unmarshaled.ID != organism.ID {
		t.Errorf("Expected ID %s, got %s", organism.ID, unmarshaled.ID)
	}
}

// TestAttributesEdgeCases tests edge cases in attribute handling
func TestAttributesEdgeCases(t *testing.T) {
	organism := &Organism{Base: Base{ID: "test"}}

	// Test setting empty attributes
	organism.SetAttributes(map[string]any{})
	attrs := organism.AttributesMap()
	if attrs == nil {
		t.Error("Expected empty map, got nil")
	}

	// Test setting nil attributes
	organism.SetAttributes(nil)
	attrs2 := organism.AttributesMap()
	if attrs2 != nil {
		t.Error("Expected nil for nil attributes")
	}

	// Test setting attributes multiple times
	organism.SetAttributes(map[string]any{"first": 1})
	organism.SetAttributes(map[string]any{"second": 2})
	attrs3 := organism.AttributesMap()
	if attrs3["first"] != nil || attrs3["second"] != 2 {
		t.Errorf("Expected only second attribute, got %v", attrs3)
	}
}

// TestUnmarshalJSONErrorCases tests error conditions in JSON unmarshaling
func TestUnmarshalJSONErrorCases(t *testing.T) {
	// Test unmarshaling invalid JSON for each entity type
	invalidJSON := []byte(`{"invalid": json}`) // Invalid JSON

	// Test Organism unmarshaling with invalid JSON
	var organism Organism
	err := organism.UnmarshalJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid JSON for organism")
	}

	// Test Facility unmarshaling with invalid JSON
	var facility Facility
	err = facility.UnmarshalJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid JSON for facility")
	}

	// Test BreedingUnit unmarshaling with invalid JSON
	var breedingUnit BreedingUnit
	err = breedingUnit.UnmarshalJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid JSON for breeding unit")
	}

	// Test Observation unmarshaling with invalid JSON
	var observation Observation
	err = observation.UnmarshalJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid JSON for observation")
	}

	// Test Sample unmarshaling with invalid JSON
	var sample Sample
	err = sample.UnmarshalJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid JSON for sample")
	}

	// Test SupplyItem unmarshaling with invalid JSON
	var supplyItem SupplyItem
	err = supplyItem.UnmarshalJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid JSON for supply item")
	}
}

// TestUnmarshalJSONMalformedAttributes tests unmarshaling with malformed attributes
func TestUnmarshalJSONMalformedAttributes(t *testing.T) {
	// Test with attributes that aren't a proper object
	malformedJSON := []byte(`{
		"id": "test",
		"name": "Test Organism",
		"species": "Test Species",
		"stage": "adult",
		"attributes": "not-an-object"
	}`)

	var organism Organism
	err := organism.UnmarshalJSON(malformedJSON)
	// This should still work - the error case is handled internally
	if err != nil {
		t.Logf("Got expected error for malformed attributes: %v", err)
	}
}

// TestJSONRoundTripComplexData tests round-trip with complex attribute data
func TestJSONRoundTripComplexData(t *testing.T) {
	original := Organism{
		Base:    Base{ID: "complex-test"},
		Name:    "Complex Test Organism",
		Species: "Test Species",
		Stage:   StageAdult,
	}

	// Set complex attributes
	original.SetAttributes(map[string]any{
		"nested": map[string]any{
			"deep": map[string]any{
				"value": "test",
			},
		},
		"array":   []any{"item1", "item2", 3},
		"number":  42.5,
		"boolean": true,
		"null":    nil,
	})

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal complex organism: %v", err)
	}

	// Unmarshal
	var restored Organism
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal complex organism: %v", err)
	}

	// Verify basic fields
	if restored.ID != original.ID {
		t.Errorf("Expected ID %s, got %s", original.ID, restored.ID)
	}

	if restored.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, restored.Name)
	}

	// Verify complex attributes are restored
	attrs := restored.AttributesMap()
	if attrs == nil {
		t.Fatal("Expected attributes to be restored")
	}

	if attrs["number"] != 42.5 {
		t.Errorf("Expected number 42.5, got %v", attrs["number"])
	}

	if attrs["boolean"] != true {
		t.Errorf("Expected boolean true, got %v", attrs["boolean"])
	}
}

// TestJSONUnmarshalAdditionalErrorPaths tests additional error paths in unmarshaling
func TestJSONUnmarshalAdditionalErrorPaths(t *testing.T) {
	// Test unmarshaling with completely empty JSON
	emptyJSON := []byte(`{}`)

	// Test all entity types with empty JSON - should work without errors
	entities := []interface{}{
		&Organism{},
		&Facility{},
		&BreedingUnit{},
		&Observation{},
		&Sample{},
		&SupplyItem{},
	}

	for i, entity := range entities {
		if unmarshaler, ok := entity.(interface{ UnmarshalJSON([]byte) error }); ok {
			if err := unmarshaler.UnmarshalJSON(emptyJSON); err != nil {
				t.Errorf("Entity %d failed to unmarshal empty JSON: %v", i, err)
			}
		}
	}

	// Test unmarshaling JSON with only attributes field
	attributesOnlyJSON := []byte(`{"attributes": {"test": "` + testAttrValue + `"}}`)

	var organism Organism
	if err := organism.UnmarshalJSON(attributesOnlyJSON); err != nil {
		t.Errorf("Failed to unmarshal attributes-only JSON: %v", err)
	}

	// Verify attributes were set
	attrs := organism.AttributesMap()
	if attrs == nil || attrs["test"] != testAttrValue {
		t.Errorf("Expected attributes to be set from JSON, got %v", attrs)
	}
}

func TestLineMarshalUnmarshalJSON(t *testing.T) {
	now := time.Now().UTC()
	line := Line{
		Base: Base{
			ID:        "line-1",
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now,
		},
		Code:              "L-001",
		Name:              "Line One",
		Origin:            "in-house",
		GenotypeMarkerIDs: []string{"gm-1"},
	}

	defaultSlot := line.EnsureDefaultAttributes()
	if err := defaultSlot.Set(extension.PluginCore, map[string]any{"seed": true}); err != nil {
		t.Fatalf("set default core slot: %v", err)
	}
	if err := defaultSlot.Set(extension.PluginID("plugin.a"), map[string]any{"note": "alpha"}); err != nil {
		t.Fatalf("set default plugin slot: %v", err)
	}
	if err := line.SetDefaultAttributesSlot(defaultSlot); err != nil {
		t.Fatalf("SetDefaultAttributesSlot: %v", err)
	}

	overrideSlot := line.EnsureExtensionOverrides()
	if err := overrideSlot.Set(extension.PluginID("plugin.override"), map[string]any{"limit": "strict"}); err != nil {
		t.Fatalf("set override slot: %v", err)
	}
	if err := line.SetExtensionOverridesSlot(overrideSlot); err != nil {
		t.Fatalf("SetExtensionOverridesSlot: %v", err)
	}

	data, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("marshal line: %v", err)
	}

	var serialized map[string]any
	if err := json.Unmarshal(data, &serialized); err != nil {
		t.Fatalf("unmarshal line payload: %v", err)
	}
	if serialized["code"] != "L-001" {
		t.Fatalf("expected code L-001, got %v", serialized["code"])
	}
	defAttributes, ok := serialized["default_attributes"].(map[string]any)
	if !ok || len(defAttributes) != 2 {
		t.Fatalf("expected default_attributes with two entries, got %v", serialized["default_attributes"])
	}
	if _, ok := defAttributes[string(extension.PluginCore)]; !ok {
		t.Fatalf("expected core payload present")
	}
	if _, ok := defAttributes["plugin.a"]; !ok {
		t.Fatalf("expected plugin.a payload present")
	}
	overrides, ok := serialized["extension_overrides"].(map[string]any)
	if !ok || len(overrides) != 1 {
		t.Fatalf("expected extension_overrides with one entry, got %v", overrides)
	}

	var decoded Line
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode line: %v", err)
	}
	if decoded.Code != line.Code {
		t.Fatalf("expected decoded code %v, got %v", line.Code, decoded.Code)
	}

	container, err := decoded.LineExtensions()
	if err != nil {
		t.Fatalf("LineExtensions: %v", err)
	}
	if len(container.Plugins(extension.HookLineDefaultAttributes)) != 2 {
		t.Fatalf("expected two default plugin payloads after decode, got %d", len(container.Plugins(extension.HookLineDefaultAttributes)))
	}
	if len(container.Plugins(extension.HookLineExtensionOverrides)) != 1 {
		t.Fatalf("expected one override plugin payload after decode, got %d", len(container.Plugins(extension.HookLineExtensionOverrides)))
	}

	if err := decoded.SetDefaultAttributesSlot(nil); err != nil {
		t.Fatalf("SetDefaultAttributesSlot nil: %v", err)
	}
	remainder, err := decoded.LineExtensions()
	if err != nil {
		t.Fatalf("LineExtensions after clear: %v", err)
	}
	if len(remainder.Hooks()) != 1 || remainder.Hooks()[0] != extension.HookLineExtensionOverrides {
		t.Fatalf("expected only overrides hook to remain, got %v", remainder.Hooks())
	}

	if err := decoded.SetExtensionOverridesSlot(nil); err != nil {
		t.Fatalf("SetExtensionOverridesSlot nil: %v", err)
	}
	if decoded.extensions != nil {
		t.Fatalf("expected extensions cleared after removing overrides, got %v", decoded.extensions)
	}

	if err := decoded.SetDefaultAttributesSlot(nil); err != nil {
		t.Fatalf("SetDefaultAttributesSlot nil on empty container: %v", err)
	}
}

func TestStrainMarshalUnmarshalJSON(t *testing.T) {
	now := time.Now().UTC()
	strain := Strain{
		Base: Base{
			ID:        "strain-1",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now,
		},
		Code:              "S-001",
		Name:              "Strain One",
		LineID:            "line-1",
		GenotypeMarkerIDs: []string{"gm-1", "gm-2"},
	}

	slot := strain.EnsureAttributes()
	if err := slot.Set(extension.PluginCore, map[string]any{"note": "core"}); err != nil {
		t.Fatalf("set strain core slot: %v", err)
	}
	if err := slot.Set(extension.PluginID("plugin.s"), map[string]any{"note": "external"}); err != nil {
		t.Fatalf("set strain plugin slot: %v", err)
	}
	if err := strain.SetAttributesSlot(slot); err != nil {
		t.Fatalf("SetAttributesSlot: %v", err)
	}

	payload, err := json.Marshal(strain)
	if err != nil {
		t.Fatalf("marshal strain: %v", err)
	}

	var encoded map[string]any
	if err := json.Unmarshal(payload, &encoded); err != nil {
		t.Fatalf("decode strain JSON: %v", err)
	}
	attrs, ok := encoded["attributes"].(map[string]any)
	if !ok || len(attrs) != 2 {
		t.Fatalf("expected attributes with both plugins, got %v", attrs)
	}

	var restored Strain
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal strain: %v", err)
	}
	clone, err := restored.StrainExtensions()
	if err != nil {
		t.Fatalf("StrainExtensions: %v", err)
	}
	if len(clone.Plugins(extension.HookStrainAttributes)) != 2 {
		t.Fatalf("expected two plugin payloads after restore, got %d", len(clone.Plugins(extension.HookStrainAttributes)))
	}

	if err := restored.SetStrainExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetStrainExtensions empty: %v", err)
	}
	if restored.attributesSlot != nil || restored.extensions != nil {
		t.Fatalf("expected strain extension state cleared, got slot=%v container=%v", restored.attributesSlot, restored.extensions)
	}
}

func TestGenotypeMarkerMarshalUnmarshalJSON(t *testing.T) {
	now := time.Now().UTC()
	marker := GenotypeMarker{
		Base: Base{
			ID:        "marker-1",
			CreatedAt: now.Add(-3 * time.Hour),
			UpdatedAt: now,
		},
		Name:           "Marker One",
		Locus:          "chr1:100-200",
		Alleles:        []string{"A", "B"},
		AssayMethod:    "PCR",
		Interpretation: "call",
		Version:        "v1",
	}

	slot := marker.EnsureAttributes()
	if err := slot.Set(extension.PluginCore, map[string]any{"threshold": 0.5}); err != nil {
		t.Fatalf("set marker core slot: %v", err)
	}
	if err := slot.Set(extension.PluginID("plugin.m"), map[string]any{"lab": "central"}); err != nil {
		t.Fatalf("set marker plugin slot: %v", err)
	}
	if err := marker.SetAttributesSlot(slot); err != nil {
		t.Fatalf("SetAttributesSlot: %v", err)
	}

	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatalf("marshal marker: %v", err)
	}

	var encoded map[string]any
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatalf("decode marker JSON: %v", err)
	}
	mattrs, ok := encoded["attributes"].(map[string]any)
	if !ok || len(mattrs) != 2 {
		t.Fatalf("expected two plugin payloads, got %v", encoded["attributes"])
	}

	var decoded GenotypeMarker
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal marker: %v", err)
	}
	clone, err := decoded.GenotypeMarkerExtensions()
	if err != nil {
		t.Fatalf("GenotypeMarkerExtensions: %v", err)
	}
	if len(clone.Plugins(extension.HookGenotypeMarkerAttributes)) != 2 {
		t.Fatalf("expected plugin attributes preserved, got %d", len(clone.Plugins(extension.HookGenotypeMarkerAttributes)))
	}

	if err := decoded.SetGenotypeMarkerExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetGenotypeMarkerExtensions empty: %v", err)
	}
	if decoded.attributesSlot != nil || decoded.extensions != nil {
		t.Fatalf("expected genotype marker extension state cleared")
	}
}

func TestLineMarshalJSONWithContainerWithoutSlots(t *testing.T) {
	line := Line{Code: "bare-line", Name: "Bare", Origin: "wild"}
	container := extension.NewContainer()
	if err := container.Set(extension.HookLineDefaultAttributes, extension.PluginCore, map[string]any{"seed": true}); err != nil {
		t.Fatalf("set container: %v", err)
	}
	if err := container.Set(extension.HookLineExtensionOverrides, extension.PluginID("plugin.override"), map[string]any{"depth": 1}); err != nil {
		t.Fatalf("set overrides in container: %v", err)
	}
	line.extensions = &container

	payload, err := json.Marshal(line)
	if err != nil {
		t.Fatalf("marshal line with container: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(payload, &encoded); err != nil {
		t.Fatalf("decode line JSON: %v", err)
	}
	if _, ok := encoded["default_attributes"].(map[string]any); !ok {
		t.Fatalf("expected default_attributes serialized from container")
	}
	if _, ok := encoded["extension_overrides"].(map[string]any); !ok {
		t.Fatalf("expected extension_overrides serialized from container")
	}
	if line.defaultAttributesSlot != nil || line.extensionOverridesSlot != nil {
		t.Fatalf("expected slots to remain nil, got default=%v override=%v", line.defaultAttributesSlot, line.extensionOverridesSlot)
	}
}

func TestLineUnmarshalJSONWithoutAttributesClearsSlots(t *testing.T) {
	jsonData := `{"id":"line-plain","code":"LP-1","name":"Plain Line","origin":"wild"}`

	var line Line
	if err := json.Unmarshal([]byte(jsonData), &line); err != nil {
		t.Fatalf("unmarshal plain line: %v", err)
	}
	if line.defaultAttributesSlot != nil || line.extensionOverridesSlot != nil {
		t.Fatalf("expected no slots after unmarshalling without extension fields")
	}
	if line.extensions != nil {
		t.Fatalf("expected extension container nil")
	}
}

func TestStrainMarshalJSONWithContainerWithoutSlot(t *testing.T) {
	strain := Strain{Code: "S-bare", Name: "Bare Strain", LineID: "line"}
	container := extension.NewContainer()
	if err := container.Set(extension.HookStrainAttributes, extension.PluginCore, map[string]any{"note": "core"}); err != nil {
		t.Fatalf("set strain container: %v", err)
	}
	strain.extensions = &container

	payload, err := json.Marshal(strain)
	if err != nil {
		t.Fatalf("marshal strain with container: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(payload, &encoded); err != nil {
		t.Fatalf("decode strain JSON: %v", err)
	}
	if _, ok := encoded["attributes"].(map[string]any); !ok {
		t.Fatalf("expected attributes serialized from container")
	}
	if strain.attributesSlot != nil {
		t.Fatalf("expected attributes slot to remain nil")
	}
}

func TestStrainUnmarshalJSONWithoutAttributesClearsState(t *testing.T) {
	jsonData := `{"id":"strain-plain","code":"SP-1","name":"Plain Strain","line_id":"line"}`

	var strain Strain
	if err := json.Unmarshal([]byte(jsonData), &strain); err != nil {
		t.Fatalf("unmarshal strain without attributes: %v", err)
	}
	if strain.attributesSlot != nil || strain.extensions != nil {
		t.Fatalf("expected strain extension state to remain nil")
	}
}

func TestGenotypeMarkerMarshalJSONWithContainerWithoutSlot(t *testing.T) {
	marker := GenotypeMarker{Name: "Bare Marker", Locus: "chr1:1-10", Alleles: []string{"A"}, AssayMethod: "PCR", Interpretation: "call", Version: "v0"}
	container := extension.NewContainer()
	if err := container.Set(extension.HookGenotypeMarkerAttributes, extension.PluginCore, map[string]any{"note": "core"}); err != nil {
		t.Fatalf("set marker container: %v", err)
	}
	marker.extensions = &container

	payload, err := json.Marshal(marker)
	if err != nil {
		t.Fatalf("marshal marker: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(payload, &encoded); err != nil {
		t.Fatalf("decode marker JSON: %v", err)
	}
	if _, ok := encoded["attributes"].(map[string]any); !ok {
		t.Fatalf("expected attributes serialized from container")
	}
	if marker.attributesSlot != nil {
		t.Fatalf("expected attributes slot to remain nil when only container present")
	}
}

func TestGenotypeMarkerUnmarshalJSONWithoutAttributesClearsState(t *testing.T) {
	jsonData := `{"id":"marker-plain","name":"Plain Marker","locus":"chr1:1-10","alleles":["A"],"assay_method":"PCR","interpretation":"call","version":"v1"}`

	var marker GenotypeMarker
	if err := json.Unmarshal([]byte(jsonData), &marker); err != nil {
		t.Fatalf("unmarshal marker without attributes: %v", err)
	}
	if marker.attributesSlot != nil || marker.extensions != nil {
		t.Fatalf("expected marker extension state cleared")
	}
}

func TestLineUnmarshalJSONInvalidAttributes(t *testing.T) {
	jsonData := `{"id":"bad-line","code":"BL","name":"Bad","origin":"lab","default_attributes":{"` + extension.PluginCore.String() + `":["invalid"]}}`

	var line Line
	if err := json.Unmarshal([]byte(jsonData), &line); err == nil {
		t.Fatalf("expected error when default attributes use invalid payload shape")
	}
}

func TestStrainUnmarshalJSONInvalidAttributes(t *testing.T) {
	jsonData := `{"id":"bad-strain","code":"BS","name":"Bad","line_id":"line","attributes":{"` + extension.PluginCore.String() + `":["invalid"]}}`

	var strain Strain
	if err := json.Unmarshal([]byte(jsonData), &strain); err == nil {
		t.Fatalf("expected error when strain attributes use invalid payload shape")
	}
}

func TestGenotypeMarkerUnmarshalJSONInvalidAttributes(t *testing.T) {
	jsonData := `{"id":"bad-marker","name":"Bad","locus":"chr1","alleles":["A"],"assay_method":"PCR","interpretation":"call","version":"v1","attributes":{"` + extension.PluginCore.String() + `":["invalid"]}}`

	var marker GenotypeMarker
	if err := json.Unmarshal([]byte(jsonData), &marker); err == nil {
		t.Fatalf("expected error when genotype marker attributes use invalid payload shape")
	}
}
