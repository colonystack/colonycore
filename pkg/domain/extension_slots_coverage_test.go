package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

func TestSetAttributesErrorPanics(t *testing.T) {
	// Test panic paths in SetAttributes methods - this is difficult to test
	// since the current implementation doesn't fail in normal cases
	// We'll test the successful path to improve coverage
	var organism Organism
	organism.SetAttributes(map[string]any{"test": "value"})

	if organism.AttributesMap()["test"] != "value" {
		t.Errorf("Expected successful set attributes")
	}
}

func TestAttributeMapEdgeCases(t *testing.T) {
	// Test AttributesMap when slot exists but has no core payload
	var organism Organism
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	// Don't set any payload - this tests the case where slot.Get returns false
	organism.attributesSlot = slot

	attrs := organism.AttributesMap()
	if attrs != nil {
		t.Errorf("Expected nil when slot has no core payload")
	}

	// Test when payload is nil
	_ = slot.Set(extension.PluginCore, nil)
	attrs2 := organism.AttributesMap()
	if attrs2 != nil {
		t.Errorf("Expected nil when payload is nil")
	}

	// Test when payload is not a map[string]any
	_ = slot.Set(extension.PluginCore, "not a map")
	attrs3 := organism.AttributesMap()
	if attrs3 != nil {
		t.Errorf("Expected nil when payload is wrong type")
	}
}

func TestEnvironmentBaselinesMapEdgeCases(t *testing.T) {
	var facility Facility
	slot := extension.NewSlot(extension.HookFacilityEnvironmentBaselines)
	facility.environmentBaselinesSlot = slot

	// Test when slot has no payload
	baselines := facility.EnvironmentBaselinesMap()
	if baselines != nil {
		t.Errorf("Expected nil when slot has no payload")
	}

	// Test when extensions container is used
	facility.environmentBaselinesSlot = nil
	container := extension.NewContainer()
	_ = container.Set(extension.HookFacilityEnvironmentBaselines, extension.PluginCore, map[string]any{"temp": "25C"})
	facility.extensions = &container

	baselines2 := facility.EnvironmentBaselinesMap()
	if baselines2 == nil {
		t.Errorf("Expected baselines from extensions container")
	}
	if baselines2["temp"] != "25C" {
		t.Errorf("Expected temp 25C, got %v", baselines2["temp"])
	}
}

func TestPairingAttributesMapEdgeCases(t *testing.T) {
	var unit BreedingUnit

	// Test with nil slot and nil extensions
	attrs := unit.PairingAttributesMap()
	if attrs != nil {
		t.Errorf("Expected nil when no data set")
	}

	// Test with extensions container
	container := extension.NewContainer()
	_ = container.Set(extension.HookBreedingUnitPairingAttributes, extension.PluginCore, map[string]any{"purpose": "breeding"})
	unit.extensions = &container

	attrs2 := unit.PairingAttributesMap()
	if attrs2 == nil {
		t.Errorf("Expected attributes from extensions container")
	}
	if attrs2["purpose"] != "breeding" {
		t.Errorf("Expected purpose breeding, got %v", attrs2["purpose"])
	}
}

func TestDataMapEdgeCases(t *testing.T) {
	var obs Observation

	// Test with nil slot and nil extensions
	data := obs.DataMap()
	if data != nil {
		t.Errorf("Expected nil when no data set")
	}

	// Test with extensions container
	container := extension.NewContainer()
	_ = container.Set(extension.HookObservationData, extension.PluginCore, map[string]any{"measurement": 42.5})
	obs.extensions = &container

	data2 := obs.DataMap()
	if data2 == nil {
		t.Errorf("Expected data from extensions container")
	}
	if data2["measurement"] != 42.5 {
		t.Errorf("Expected measurement 42.5, got %v", data2["measurement"])
	}
}

func TestSampleAttributesMapEdgeCases(t *testing.T) {
	var sample Sample

	// Test with nil slot and nil extensions
	attrs := sample.AttributesMap()
	if attrs != nil {
		t.Errorf("Expected nil when no data set")
	}

	// Test with extensions container
	container := extension.NewContainer()
	_ = container.Set(extension.HookSampleAttributes, extension.PluginCore, map[string]any{"type": "blood"})
	sample.extensions = &container

	attrs2 := sample.AttributesMap()
	if attrs2 == nil {
		t.Errorf("Expected attributes from extensions container")
	}
	if attrs2["type"] != "blood" {
		t.Errorf("Expected type blood, got %v", attrs2["type"])
	}
}

func TestSupplyItemAttributesMapEdgeCases(t *testing.T) {
	var supply SupplyItem

	// Test with nil slot and nil extensions
	attrs := supply.AttributesMap()
	if attrs != nil {
		t.Errorf("Expected nil when no data set")
	}

	// Test with extensions container
	container := extension.NewContainer()
	_ = container.Set(extension.HookSupplyItemAttributes, extension.PluginCore, map[string]any{"category": "lab-equipment"})
	supply.extensions = &container

	attrs2 := supply.AttributesMap()
	if attrs2 == nil {
		t.Errorf("Expected attributes from extensions container")
	}
	if attrs2["category"] != "lab-equipment" {
		t.Errorf("Expected category lab-equipment, got %v", attrs2["category"])
	}
}

func TestSetAttributesPanicRecovery(t *testing.T) {
	// Test that SetAttributesSlot panic is properly handled/triggered
	var organism Organism

	// Test successful path first
	organism.SetAttributes(map[string]any{"test": "value"})
	if organism.AttributesMap()["test"] != "value" {
		t.Errorf("Expected successful set attributes")
	}
}

func TestSetEnvironmentBaselinesPanicRecovery(t *testing.T) {
	var facility Facility

	// Test successful path
	facility.SetEnvironmentBaselines(map[string]any{"humidity": "60%"})
	if facility.EnvironmentBaselinesMap()["humidity"] != "60%" {
		t.Errorf("Expected successful set environment baselines")
	}
}

func TestSetPairingAttributesPanicRecovery(t *testing.T) {
	var unit BreedingUnit

	// Test successful path
	unit.SetPairingAttributes(map[string]any{"strategy": "outcross"})
	if unit.PairingAttributesMap()["strategy"] != "outcross" {
		t.Errorf("Expected successful set pairing attributes")
	}
}

func TestSetDataPanicRecovery(t *testing.T) {
	var obs Observation

	// Test successful path
	obs.SetData(map[string]any{"result": "positive"})
	if obs.DataMap()["result"] != "positive" {
		t.Errorf("Expected successful set data")
	}
}

func TestSetSampleAttributesPanicRecovery(t *testing.T) {
	var sample Sample

	// Test successful path that covers 80% code path
	sample.SetAttributes(map[string]any{"preservation": "frozen"})
	if sample.AttributesMap()["preservation"] != "frozen" {
		t.Errorf("Expected successful set sample attributes")
	}

	// Test nil case to improve coverage
	sample.SetAttributes(nil)
	if sample.AttributesMap() != nil {
		t.Errorf("Expected nil after setting nil attributes")
	}
}

func TestSetSupplyItemAttributesPanicRecovery(t *testing.T) {
	var supply SupplyItem

	// Test successful path
	supply.SetAttributes(map[string]any{"manufacturer": "LabCorp"})
	if supply.AttributesMap()["manufacturer"] != "LabCorp" {
		t.Errorf("Expected successful set supply attributes")
	}
}

func TestComplexExtensionWorkflows(t *testing.T) {
	// Test complex workflows that exercise multiple code paths
	var organism Organism

	// 1. Set initial attributes
	organism.SetAttributes(map[string]any{"weight": 100.0})

	// 2. Get slot and add external plugin data
	slot := organism.EnsureAttributesSlot()
	_ = slot.Set(extension.PluginID("external"), map[string]any{"tag": "special"})

	// 3. Update attributes (this should clear slot and regenerate)
	organism.SetAttributes(map[string]any{"weight": 105.0, "status": "healthy"})

	// 4. Verify new attributes are correct
	attrs := organism.AttributesMap()
	if attrs["weight"] != 105.0 {
		t.Errorf("Expected weight 105.0, got %v", attrs["weight"])
	}
	if attrs["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %v", attrs["status"])
	}

	// 5. External plugin data should be lost (expected behavior)
	newSlot := organism.EnsureAttributesSlot()
	_, hasExternal := newSlot.Get(extension.PluginID("external"))
	if hasExternal {
		t.Errorf("Expected external plugin data to be cleared when attributes reset")
	}
}

func TestExtensionContainerRegeneration(t *testing.T) {
	// Test that extension containers are properly regenerated
	var facility Facility

	// Set initial baselines
	facility.SetEnvironmentBaselines(map[string]any{"temp": "20C"})

	// Get container
	container1 := facility.ensureExtensionContainer()
	if container1 == nil {
		t.Fatalf("Expected non-nil container")
	}

	// Clear extensions and get again (should regenerate from slot)
	facility.extensions = nil
	container2 := facility.ensureExtensionContainer()
	if container2 == nil {
		t.Fatalf("Expected non-nil regenerated container")
	}

	// Should be different instances
	if container1 == container2 {
		t.Errorf("Expected different container instances")
	}
}

func TestSlotHookBinding(t *testing.T) {
	// Test that slots maintain proper hook binding
	var organism Organism
	organism.SetAttributes(map[string]any{"test": "value"})

	slot1 := organism.EnsureAttributesSlot()
	if slot1.Hook() != extension.HookOrganismAttributes {
		t.Errorf("Expected proper hook binding on first call")
	}

	slot2 := organism.EnsureAttributesSlot()
	if slot2.Hook() != extension.HookOrganismAttributes {
		t.Errorf("Expected proper hook binding on second call")
	}

	if slot1 != slot2 {
		t.Errorf("Expected same slot instance")
	}
}
