package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

func TestExtensionContainerCoverageOrganism(t *testing.T) {
	// Test ensureExtensionContainer when extensions is nil
	var organism Organism
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"test": testAttrValue}))

	container := organism.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil extension container")
	}

	// Test when extensions is already set
	secondCall := organism.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var organism2 Organism
	if organism2.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh organism")
	}
}

func TestExtensionContainerCoverageFacility(t *testing.T) {
	// Test ensureExtensionContainer when extensions is nil
	var facility Facility
	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(map[string]any{"temp": "22C"}))

	container := facility.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil extension container")
	}

	// Test when extensions is already set
	secondCall := facility.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var facility2 Facility
	if facility2.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh facility")
	}
}

func TestExtensionContainerCoverageBreedingUnit(t *testing.T) {
	// Test ensureExtensionContainer when extensions is nil
	var unit BreedingUnit
	mustNoError(t, "ApplyPairingAttributes", unit.ApplyPairingAttributes(map[string]any{"purpose": "research"}))

	container := unit.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil extension container")
	}

	// Test when extensions is already set
	secondCall := unit.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var unit2 BreedingUnit
	if unit2.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh breeding unit")
	}
}

func TestExtensionContainerCoverageObservation(t *testing.T) {
	// Test ensureExtensionContainer when extensions is nil
	var obs Observation
	mustNoError(t, "ApplyObservationData", obs.ApplyObservationData(map[string]any{"weight": 50.5}))

	container := obs.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil extension container")
	}

	// Test when extensions is already set
	secondCall := obs.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var obs2 Observation
	if obs2.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh observation")
	}
}

func TestExtensionContainerCoverageSample(t *testing.T) {
	// Test ensureExtensionContainer when extensions is nil
	var sample Sample
	mustNoError(t, "ApplySampleAttributes", sample.ApplySampleAttributes(map[string]any{"volume": "5ml"}))

	container := sample.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil extension container")
	}

	// Test when extensions is already set
	secondCall := sample.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var sample2 Sample
	if sample2.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh sample")
	}
}

func TestExtensionContainerCoverageSupplyItem(t *testing.T) {
	// Test ensureExtensionContainer when extensions is nil
	var supply SupplyItem
	mustNoError(t, "ApplySupplyAttributes", supply.ApplySupplyAttributes(map[string]any{"brand": "TestBrand"}))

	container := supply.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil extension container")
	}

	// Test when extensions is already set
	secondCall := supply.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var supply2 SupplyItem
	if supply2.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh supply")
	}
}

func TestExtensionContainerCoverageLine(t *testing.T) {
	var line Line
	mustNoError(t, "ApplyDefaultAttributes", line.ApplyDefaultAttributes(map[string]any{
		extension.PluginCore.String(): map[string]any{"seed": true},
	}))
	container := line.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil line extension container")
	}

	secondCall := line.ensureExtensionContainer()
	if secondCall != container {
		t.Errorf("Expected same container instance on second call")
	}

	var empty Line
	container2 := empty.ensureExtensionContainer()
	if container2 == nil {
		t.Fatalf("Expected non-nil container even with nil slots")
	}
}

func TestExtensionContainerCoverageStrain(t *testing.T) {
	var strain Strain
	mustNoError(t, "ApplyStrainAttributes", strain.ApplyStrainAttributes(map[string]any{
		extension.PluginCore.String(): map[string]any{"label": "strain"},
	}))
	container := strain.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil strain extension container")
	}

	second := strain.ensureExtensionContainer()
	if second != container {
		t.Errorf("Expected same container instance on subsequent call")
	}

	var empty Strain
	if empty.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh strain")
	}
}

func TestExtensionContainerCoverageGenotypeMarker(t *testing.T) {
	var marker GenotypeMarker
	mustNoError(t, "ApplyGenotypeMarkerAttributes", marker.ApplyGenotypeMarkerAttributes(map[string]any{
		extension.PluginCore.String(): map[string]any{"label": "geno"},
	}))
	container := marker.ensureExtensionContainer()
	if container == nil {
		t.Fatalf("Expected non-nil genotype extension container")
	}

	second := marker.ensureExtensionContainer()
	if second != container {
		t.Errorf("Expected same container instance on subsequent call")
	}

	var empty GenotypeMarker
	if empty.ensureExtensionContainer() == nil {
		t.Fatalf("Expected non-nil container for fresh genotype marker")
	}
}

func TestSlotFromContainerEdgeCases(t *testing.T) {
	// Test with container that has empty plugins
	container := extension.NewContainer()
	slot := slotFromContainer(extension.HookOrganismAttributes, &container)
	if slot == nil {
		t.Fatalf("Expected non-nil slot")
	}
	if slot.Hook() != extension.HookOrganismAttributes {
		t.Errorf("Expected hook to be set correctly")
	}
	if len(slot.Plugins()) != 0 {
		t.Errorf("Expected no plugins for empty container")
	}

	// Test with container that has multiple plugins
	_ = container.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"core": true})
	_ = container.Set(extension.HookOrganismAttributes, extension.PluginID("external"), map[string]any{"external": true})

	slot2 := slotFromContainer(extension.HookOrganismAttributes, &container)
	if len(slot2.Plugins()) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(slot2.Plugins()))
	}

	// Test error case where slot.Get returns false
	slot3 := extension.NewSlot(extension.HookOrganismAttributes)
	// Manually create a slot that will return false for Get (this is edge case testing)
	_ = slot3.Set(extension.PluginCore, map[string]any{"test": testAttrValue})

	// Create container from this slot
	container2, err := containerFromSlot(extension.HookOrganismAttributes, slot3)
	if err != nil {
		t.Fatalf("containerFromSlot failed: %v", err)
	}
	if container2 == nil {
		t.Fatalf("Expected non-nil container")
	}
}

func TestContainerFromSlotErrorHandling(t *testing.T) {
	// Test with slot that has error on Set
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	_ = slot.Set(extension.PluginCore, map[string]any{"test": testAttrValue})

	// Try to create container with wrong hook (this should work but test edge cases)
	container, err := containerFromSlot(extension.HookOrganismAttributes, slot)
	if err != nil {
		t.Fatalf("containerFromSlot failed: %v", err)
	}
	if container == nil {
		t.Fatalf("Expected non-nil container")
	}
}

func TestExtensionContainerSlotInteraction(t *testing.T) {
	// Test complex interaction between slots and containers
	var organism Organism

	// Start with attributes
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"initial": testAttrValue}))
	container := organism.ensureExtensionContainer()
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginID("external"), map[string]any{"external": "data"}); err != nil {
		t.Fatalf("set external payload: %v", err)
	}

	cloned, err := organism.OrganismExtensions()
	if err != nil {
		t.Fatalf("OrganismExtensions: %v", err)
	}
	var dup Organism
	if err := dup.SetOrganismExtensions(cloned); err != nil {
		t.Fatalf("SetOrganismExtensions: %v", err)
	}

	// Verify container contains both core and external data
	coreData, hasCore := dup.ensureExtensionContainer().Get(extension.HookOrganismAttributes, extension.PluginCore)
	if !hasCore {
		t.Errorf("Expected core plugin data in container")
	}
	if coreMap := coreData.(map[string]any); coreMap["initial"] != testAttrValue {
		t.Errorf("Expected core data to be preserved")
	}

	externalData, hasExternal := dup.ensureExtensionContainer().Get(extension.HookOrganismAttributes, extension.PluginID("external"))
	if !hasExternal {
		t.Errorf("Expected external plugin data in container")
	}
	if extMap := externalData.(map[string]any); extMap["external"] != "data" {
		t.Errorf("Expected external data to be preserved")
	}
}

func TestSetAttributesNilHandling(t *testing.T) {
	// Test SetAttributes with nil for various entities
	var organism Organism
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"test": testAttrValue}))

	// Verify it's set
	if organism.CoreAttributes() == nil {
		t.Fatalf("Expected attributes to be set")
	}

	// Set to nil
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(nil))
	if organism.CoreAttributes() != nil {
		t.Errorf("Expected attributes to be nil after setting nil")
	}
	assertContainerEmpty(t, organism.extensions, "expected extensions to be cleared")
}

func TestSetEnvironmentBaselinesNilHandling(t *testing.T) {
	var facility Facility
	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(map[string]any{"temp": "22C"}))

	// Verify it's set
	if facility.EnvironmentBaselines() == nil {
		t.Fatalf("Expected baselines to be set")
	}

	// Set to nil
	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(nil))
	if facility.EnvironmentBaselines() != nil {
		t.Errorf("Expected baselines to be nil after setting nil")
	}
	assertContainerEmpty(t, facility.extensions, "expected extensions to be cleared")
}

func TestSetPairingAttributesNilHandling(t *testing.T) {
	var unit BreedingUnit
	mustNoError(t, "ApplyPairingAttributes", unit.ApplyPairingAttributes(map[string]any{"purpose": "test"}))

	// Verify it's set
	if unit.PairingAttributes() == nil {
		t.Fatalf("Expected pairing attributes to be set")
	}

	// Set to nil
	mustNoError(t, "ApplyPairingAttributes", unit.ApplyPairingAttributes(nil))
	if unit.PairingAttributes() != nil {
		t.Errorf("Expected pairing attributes to be nil after setting nil")
	}
	assertContainerEmpty(t, unit.extensions, "expected extensions to be cleared")
}

func TestSetDataNilHandling(t *testing.T) {
	var obs Observation
	mustNoError(t, "ApplyObservationData", obs.ApplyObservationData(map[string]any{"weight": 50.5}))

	// Verify it's set
	if obs.ObservationData() == nil {
		t.Fatalf("Expected data to be set")
	}

	// Set to nil
	mustNoError(t, "ApplyObservationData", obs.ApplyObservationData(nil))
	if obs.ObservationData() != nil {
		t.Errorf("Expected data to be nil after setting nil")
	}
	assertContainerEmpty(t, obs.extensions, "expected extensions to be cleared")
}

func TestMapAccessorsWithNilExtensions(t *testing.T) {
	// Test CoreAttributes when both slot and extensions are nil
	var organism Organism
	attrs := organism.CoreAttributes()
	if attrs != nil {
		t.Errorf("Expected nil attributes map when no data set")
	}

	// Test when extensions exist but slot is nil
	organism.extensions = &extension.Container{}
	attrs2 := organism.CoreAttributes()
	if attrs2 != nil {
		t.Errorf("Expected nil attributes when container has no data")
	}
}

// TestContainerPluginMethods tests plugin management in containers
func TestContainerPluginMethods(t *testing.T) {
	organism := &Organism{Base: Base{ID: "test"}}
	container := organism.ensureExtensionContainer()

	// Test plugins method when no data exists
	plugins := container.Plugins(extension.HookOrganismAttributes)
	if len(plugins) != 0 {
		t.Errorf("Expected empty plugins list, got %v", plugins)
	}

	// Add some plugin data
	// Test plugins method returns sorted list
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"test": 1}); err != nil {
		t.Fatalf("unexpected Set error: %v", err)
	}
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginID("custom"), map[string]any{"test": 2}); err != nil {
		t.Fatalf("unexpected Set error: %v", err)
	}

	plugins = container.Plugins(extension.HookOrganismAttributes)
	if len(plugins) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(plugins))
	}

	// Should be sorted alphabetically
	if plugins[0] != extension.PluginCore || plugins[1] != extension.PluginID("custom") {
		t.Errorf("Expected sorted plugins [core, custom], got %v", plugins)
	}
}

// TestAllContainerTypes tests all entity types have working containers
func TestAllContainerTypes(t *testing.T) {
	entities := []struct {
		name string
		test func() *extension.Container
	}{
		{"Organism", func() *extension.Container {
			o := &Organism{Base: Base{ID: "test"}}
			return o.ensureExtensionContainer()
		}},
		{"Facility", func() *extension.Container {
			f := &Facility{Base: Base{ID: "test"}}
			return f.ensureExtensionContainer()
		}},
		{"BreedingUnit", func() *extension.Container {
			b := &BreedingUnit{Base: Base{ID: "test"}}
			return b.ensureExtensionContainer()
		}},
		{"Observation", func() *extension.Container {
			o := &Observation{Base: Base{ID: "test"}}
			return o.ensureExtensionContainer()
		}},
		{"Sample", func() *extension.Container {
			s := &Sample{Base: Base{ID: "test"}}
			return s.ensureExtensionContainer()
		}},
		{"SupplyItem", func() *extension.Container {
			s := &SupplyItem{Base: Base{ID: "test"}}
			return s.ensureExtensionContainer()
		}},
	}

	for _, entity := range entities {
		t.Run(entity.name, func(t *testing.T) {
			container := entity.test()
			if container == nil {
				t.Errorf("%s ensureExtensionContainer() returned nil", entity.name)
			}
		})
	}
}

// TestEnsureExtensionContainerWithExistingData tests container initialization with existing data
func TestEnsureExtensionContainerWithExistingData(t *testing.T) {
	// Test Organism with existing attributes
	organism := &Organism{Base: Base{ID: "test"}}
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"initial": "data"}))

	container := organism.ensureExtensionContainer()
	if container == nil {
		t.Fatal("Expected non-nil container")
	}

	// Verify existing data is preserved
	data, ok := container.Get(extension.HookOrganismAttributes, extension.PluginCore)
	if !ok || data == nil {
		t.Error("Expected existing attributes to be preserved in container")
	}

	// Test Facility with existing environment baselines
	facility := &Facility{Base: Base{ID: "test"}}
	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(map[string]any{"temp": "20C"}))

	facilityContainer := facility.ensureExtensionContainer()
	envData, envOk := facilityContainer.Get(extension.HookFacilityEnvironmentBaselines, extension.PluginCore)
	if !envOk || envData == nil {
		t.Error("Expected existing environment baselines to be preserved")
	}

	// Test BreedingUnit with existing pairing attributes
	breedingUnit := &BreedingUnit{Base: Base{ID: "test"}}
	mustNoError(t, "ApplyPairingAttributes", breedingUnit.ApplyPairingAttributes(map[string]any{"pairing": "data"}))

	breedingContainer := breedingUnit.ensureExtensionContainer()
	pairingData, pairingOk := breedingContainer.Get(extension.HookBreedingUnitPairingAttributes, extension.PluginCore)
	if !pairingOk || pairingData == nil {
		t.Error("Expected existing pairing attributes to be preserved")
	}

	// Test Observation with existing data
	observation := &Observation{Base: Base{ID: "test"}}
	mustNoError(t, "ApplyObservationData", observation.ApplyObservationData(map[string]any{"measurement": testAttrValue}))

	obsContainer := observation.ensureExtensionContainer()
	obsData, obsOk := obsContainer.Get(extension.HookObservationData, extension.PluginCore)
	if !obsOk || obsData == nil {
		t.Error("Expected existing observation data to be preserved")
	}

	// Test Sample with existing attributes
	sample := &Sample{Base: Base{ID: "test"}}
	mustNoError(t, "ApplySampleAttributes", sample.ApplySampleAttributes(map[string]any{"sample": "data"}))

	sampleContainer := sample.ensureExtensionContainer()
	sampleData, sampleOk := sampleContainer.Get(extension.HookSampleAttributes, extension.PluginCore)
	if !sampleOk || sampleData == nil {
		t.Error("Expected existing sample attributes to be preserved")
	}

	// Test SupplyItem with existing attributes
	supplyItem := &SupplyItem{Base: Base{ID: "test"}}
	mustNoError(t, "ApplySupplyAttributes", supplyItem.ApplySupplyAttributes(map[string]any{"supply": "data"}))

	supplyContainer := supplyItem.ensureExtensionContainer()
	supplyData, supplyOk := supplyContainer.Get(extension.HookSupplyItemAttributes, extension.PluginCore)
	if !supplyOk || supplyData == nil {
		t.Error("Expected existing supply item attributes to be preserved")
	}
}

// TestEnsureExtensionContainerNilSlots tests container creation when slots are nil
func TestEnsureExtensionContainerNilSlots(t *testing.T) {
	// Test all entity types with completely empty extension state
	organism := &Organism{Base: Base{ID: "test"}}

	container := organism.ensureExtensionContainer()
	if container == nil {
		t.Error("Expected non-nil container even with no extension data")
	}

	// Test that we can use the container
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"test": testAttrValue}); err != nil {
		t.Fatalf("Expected container with data to set successfully: %v", err)
	}

	data, ok := container.Get(extension.HookOrganismAttributes, extension.PluginCore)
	if !ok || data == nil {
		t.Error("Expected to be able to use container after creation from nil slot")
	}
}
