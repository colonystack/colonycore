package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

const testMutated = "mutated"

func TestOrganismSetAttributesClonesInputAndOutput(t *testing.T) {
	original := map[string]any{
		"nested": []any{map[string]any{"k": "v"}},
	}
	var organism Organism
	organism.SetAttributes(original)

	// Mutate original input; stored value should remain unchanged.
	original["nested"].([]any)[0].(map[string]any)["k"] = testMutated
	if organism.AttributesMap()["nested"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored attributes to remain unchanged")
	}

	// Mutate returned map; stored value should remain unchanged.
	view := organism.AttributesMap()
	view["nested"].([]any)[0].(map[string]any)["k"] = "changed"
	if organism.AttributesMap()["nested"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored attributes to remain unchanged after view mutation")
	}
}

func TestFacilityEnvironmentBaselinesHelpersClone(t *testing.T) {
	input := map[string]any{"temp": []int{20}}

	var facility Facility
	facility.SetEnvironmentBaselines(input)

	input["temp"].([]int)[0] = 99
	if facility.EnvironmentBaselinesMap()["temp"].([]int)[0] != 20 {
		t.Fatalf("expected facility baselines to be cloned from input")
	}

	baselines := facility.EnvironmentBaselinesMap()
	baselines["temp"].([]int)[0] = 30
	if facility.EnvironmentBaselinesMap()["temp"].([]int)[0] != 20 {
		t.Fatalf("expected facility baselines to remain unchanged after view mutation")
	}
}

func TestSampleSetAttributesNil(t *testing.T) {
	var sample Sample
	sample.SetAttributes(nil)
	if sample.AttributesMap() != nil {
		t.Fatalf("expected nil attributes when input is nil")
	}
	if sample.AttributesMap() != nil {
		t.Fatalf("expected nil attribute map when stored map is nil")
	}
}

func TestOrganismAttributesSlotBridgeRoundTrip(t *testing.T) {
	organism := Organism{}
	organism.SetAttributes(map[string]any{"flag": true})

	slot := organism.EnsureAttributesSlot()
	if slot.Hook() != extension.HookOrganismAttributes {
		t.Fatalf("expected organism slot hook %q, got %q", extension.HookOrganismAttributes, slot.Hook())
	}

	corePayload, ok := slot.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected core payload to be present")
	}
	payload, ok := corePayload.(map[string]any)
	if !ok || payload["flag"] != true {
		t.Fatalf("unexpected payload retrieved from slot: %+v", corePayload)
	}

	payload["flag"] = false
	if organism.AttributesMap()["flag"] != true {
		t.Fatalf("expected slot payload clone to avoid mutating organism attributes")
	}

	update := map[string]any{"flag": false}
	if err := slot.Set(extension.PluginCore, update); err != nil {
		t.Fatalf("unexpected error setting slot payload: %v", err)
	}

	update["flag"] = true
	if err := organism.SetAttributesSlot(slot); err != nil {
		t.Fatalf("unexpected error applying slot payload: %v", err)
	}

	if value, ok := organism.AttributesMap()["flag"].(bool); !ok || value {
		t.Fatalf("expected organism attributes to be updated from slot")
	}
}

func TestExtensionSlotBridgeRoundTripsOtherEntities(t *testing.T) {
	tempMap := map[string]any{"temp": "22C"}
	env := Facility{}
	env.SetEnvironmentBaselines(tempMap)
	envSlot := env.EnsureEnvironmentBaselinesSlot()
	if envSlot.Hook() != extension.HookFacilityEnvironmentBaselines {
		t.Fatalf("expected facility hook %q", extension.HookFacilityEnvironmentBaselines)
	}
	payload := map[string]any{"temp": "25C"}
	_ = envSlot.Set(extension.PluginCore, payload)
	payload["temp"] = testMutated
	if err := env.SetEnvironmentBaselinesSlot(envSlot); err != nil {
		t.Fatalf("unexpected error setting facility slot: %v", err)
	}
	if env.EnvironmentBaselinesMap()["temp"] != "25C" {
		t.Fatalf("expected facility baselines to reflect slot payload")
	}

	breeding := BreedingUnit{}
	breeding.SetPairingAttributes(map[string]any{"note": "initial"})
	bSlot := breeding.EnsurePairingAttributesSlot()
	if bSlot.Hook() != extension.HookBreedingUnitPairingAttributes {
		t.Fatalf("expected breeding unit hook %q", extension.HookBreedingUnitPairingAttributes)
	}
	_ = bSlot.Set(extension.PluginCore, map[string]any{"note": "updated"})
	if err := breeding.SetPairingAttributesSlot(bSlot); err != nil {
		t.Fatalf("unexpected error applying breeding slot: %v", err)
	}
	if breeding.PairingAttributesMap()["note"] != "updated" {
		t.Fatalf("expected breeding attributes to reflect slot payload")
	}

	observation := Observation{}
	observation.SetData(map[string]any{"metric": 3})
	oSlot := observation.EnsureObservationDataSlot()
	if oSlot.Hook() != extension.HookObservationData {
		t.Fatalf("expected observation hook %q", extension.HookObservationData)
	}
	_ = oSlot.Set(extension.PluginCore, map[string]any{"metric": 4})
	if err := observation.SetObservationDataSlot(oSlot); err != nil {
		t.Fatalf("unexpected error applying observation slot: %v", err)
	}
	if observation.DataMap()["metric"] != 4 {
		t.Fatalf("expected observation data to reflect slot payload")
	}

	sample := Sample{}
	sample.SetAttributes(map[string]any{"batch": "A"})
	sSlot := sample.EnsureSampleAttributesSlot()
	if sSlot.Hook() != extension.HookSampleAttributes {
		t.Fatalf("expected sample hook %q", extension.HookSampleAttributes)
	}
	_ = sSlot.Set(extension.PluginCore, map[string]any{"batch": "B"})
	if err := sample.SetSampleAttributesSlot(sSlot); err != nil {
		t.Fatalf("unexpected error applying sample slot: %v", err)
	}
	if sample.AttributesMap()["batch"] != "B" {
		t.Fatalf("expected sample attributes to reflect slot payload")
	}

	supply := SupplyItem{}
	supply.SetAttributes(map[string]any{"reorder": true})
	uSlot := supply.EnsureSupplyItemAttributesSlot()
	if uSlot.Hook() != extension.HookSupplyItemAttributes {
		t.Fatalf("expected supply hook %q", extension.HookSupplyItemAttributes)
	}
	_ = uSlot.Set(extension.PluginCore, map[string]any{"reorder": false})
	if err := supply.SetSupplyItemAttributesSlot(uSlot); err != nil {
		t.Fatalf("unexpected error applying supply slot: %v", err)
	}
	if supply.AttributesMap()["reorder"] != false {
		t.Fatalf("expected supply attributes to reflect slot payload")
	}
}

func TestExtensionSlotBridgeRejectsNonCorePlugins(t *testing.T) {
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	if err := slot.Set(extension.PluginID("external.plugin"), map[string]any{"flag": true}); err != nil {
		t.Fatalf("expected slot set to succeed, got %v", err)
	}
	var organism Organism
	if err := organism.SetAttributesSlot(slot); err == nil {
		t.Fatalf("expected non-core payload to be rejected")
	}
	if organism.AttributesMap() != nil {
		t.Fatalf("expected organism attributes to remain nil when payload rejected")
	}
}

func TestExtensionSlotBridgeRejectsNonObjectPayload(t *testing.T) {
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	if err := slot.Set(extension.PluginCore, "scalar"); err != nil {
		t.Fatalf("expected to set scalar payload for test: %v", err)
	}
	var organism Organism
	if err := organism.SetAttributesSlot(slot); err == nil {
		t.Fatalf("expected scalar payload to be rejected")
	}
}

func TestExtensionSlotBridgeNilSlotHandling(t *testing.T) {
	var sample Sample
	if slot := sample.EnsureSampleAttributesSlot(); slot == nil {
		t.Fatalf("expected non-nil slot")
	}
	if err := sample.SetSampleAttributesSlot(nil); err != nil {
		t.Fatalf("expected nil slot to be accepted: %v", err)
	}
	if sample.AttributesMap() != nil {
		t.Fatalf("expected attributes to remain nil when setting nil slot")
	}
}

func TestLineAndStrainEnsureSlots(t *testing.T) {
	line := &Line{}
	defaultSlot := line.EnsureDefaultAttributes()
	if defaultSlot == nil {
		t.Fatalf("expected default attributes slot")
	}
	if defaultSlot.Hook() != extension.HookLineDefaultAttributes {
		t.Fatalf("expected line default hook %q, got %q", extension.HookLineDefaultAttributes, defaultSlot.Hook())
	}
	if err := defaultSlot.Set(extension.PluginCore, map[string]any{"k": "v"}); err != nil {
		t.Fatalf("set default slot: %v", err)
	}
	if line.EnsureDefaultAttributes() != defaultSlot {
		t.Fatalf("expected EnsureDefaultAttributes to return same slot pointer")
	}

	overrideSlot := line.EnsureExtensionOverrides()
	if overrideSlot == nil {
		t.Fatalf("expected overrides slot")
	}
	if overrideSlot.Hook() != extension.HookLineExtensionOverrides {
		t.Fatalf("expected overrides hook %q, got %q", extension.HookLineExtensionOverrides, overrideSlot.Hook())
	}
	if err := overrideSlot.Set(extension.PluginCore, map[string]any{"override": true}); err != nil {
		t.Fatalf("set override slot: %v", err)
	}
	if line.EnsureExtensionOverrides() != overrideSlot {
		t.Fatalf("expected EnsureExtensionOverrides to be idempotent")
	}

	strain := &Strain{}
	strainSlot := strain.EnsureAttributes()
	if strainSlot.Hook() != extension.HookStrainAttributes {
		t.Fatalf("expected strain hook %q", extension.HookStrainAttributes)
	}

	genotype := &GenotypeMarker{}
	genotypeSlot := genotype.EnsureAttributes()
	if genotypeSlot.Hook() != extension.HookGenotypeMarkerAttributes {
		t.Fatalf("expected genotype marker hook %q", extension.HookGenotypeMarkerAttributes)
	}

	// Simulate JSON unmarshalling that loses hook binding.
	unboundSlot := extension.NewSlot("")
	strain.Attributes = unboundSlot
	genotype.Attributes = unboundSlot.Clone()
	if strain.EnsureAttributes().Hook() != extension.HookStrainAttributes {
		t.Fatalf("expected EnsureAttributes to rebind strain hook")
	}
	if genotype.EnsureAttributes().Hook() != extension.HookGenotypeMarkerAttributes {
		t.Fatalf("expected EnsureAttributes to rebind genotype hook")
	}
}

func TestAssignAndCloneExtensionMap(t *testing.T) {
	values := map[string]any{
		"slice": []int{1, 2},
		"map":   map[string]any{"flag": true},
	}
	cloned := cloneExtensionMap(values)
	if cloned["slice"].([]int)[0] != 1 {
		t.Fatalf("expected slice clone to retain values")
	}
	cloned["slice"].([]int)[0] = 99
	if values["slice"].([]int)[0] != 1 {
		t.Fatalf("expected clone to be independent of original")
	}

	assigned := assignExtensionMap(values)
	if assigned["map"].(map[string]any)["flag"] != true {
		t.Fatalf("expected assigned clone to retain flag true")
	}
	assigned["map"].(map[string]any)["flag"] = false
	if values["map"].(map[string]any)["flag"] != true {
		t.Fatalf("expected assigned map to be independent")
	}

	if cloneExtensionMap(nil) != nil {
		t.Fatalf("expected nil clone to remain nil")
	}
	if assignExtensionMap(nil) != nil {
		t.Fatalf("expected nil assign to remain nil")
	}
}
