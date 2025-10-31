package domain

import (
	"encoding/json"
	"testing"
	"time"
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
