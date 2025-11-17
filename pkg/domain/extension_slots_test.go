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
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(original))

	// Mutate original input; stored value should remain unchanged.
	original["nested"].([]any)[0].(map[string]any)["k"] = testMutated
	if organism.CoreAttributes()["nested"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored attributes to remain unchanged")
	}

	// Mutate returned map; stored value should remain unchanged.
	view := organism.CoreAttributes()
	view["nested"].([]any)[0].(map[string]any)["k"] = "changed"
	if organism.CoreAttributes()["nested"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored attributes to remain unchanged after view mutation")
	}
}

func TestOrganismEnsureAttributesSlotCaching(t *testing.T) {
	var organism Organism
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"flag": true}))

	first := organism.EnsureAttributesSlot()
	if first == nil {
		t.Fatalf("expected cached slot instance")
	}
	if organism.attributesSlot != first {
		t.Fatalf("expected slot to be cached on organism")
	}

	second := organism.EnsureAttributesSlot()
	if second != first {
		t.Fatalf("expected repeated ensure to reuse cached slot")
	}

	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"flag": false}))
	replacement := organism.EnsureAttributesSlot()
	if replacement == first {
		t.Fatalf("expected ensure to build fresh slot after reset")
	}
	payload, ok := replacement.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected payload on replacement slot")
	}
	if payload.(map[string]any)["flag"] != false {
		t.Fatalf("expected replacement slot to reflect updated map")
	}
}

func TestOrganismSetAttributesSlotClonesInput(t *testing.T) {
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	input := map[string]any{"flag": true}
	if err := slot.Set(extension.PluginCore, input); err != nil {
		t.Fatalf("set payload: %v", err)
	}

	var organism Organism
	if err := organism.SetAttributesSlot(slot); err != nil {
		t.Fatalf("SetAttributesSlot: %v", err)
	}
	if organism.attributesSlot == slot {
		t.Fatalf("expected organism to store slot clone, not original pointer")
	}
	if organism.extensions == nil {
		t.Fatalf("expected organism extension container to be initialised")
	}

	// Mutate original inputs; cached slot should remain unchanged.
	input["flag"] = false
	if err := slot.Set(extension.PluginCore, map[string]any{"flag": "mutated"}); err != nil {
		t.Fatalf("mutate original slot: %v", err)
	}

	cached := organism.EnsureAttributesSlot()
	payload, ok := cached.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected cached payload")
	}
	if payload.(map[string]any)["flag"] != true {
		t.Fatalf("expected cached slot to remain unchanged after input mutation")
	}

	if err := organism.SetAttributesSlot(nil); err != nil {
		t.Fatalf("SetAttributesSlot nil: %v", err)
	}
	if organism.CoreAttributes() != nil {
		t.Fatalf("expected attributes view to clear when setting nil")
	}
	assertSlotEmpty(t, organism.attributesSlot, "expected slot to be cleared when setting nil")
	assertContainerEmpty(t, organism.extensions, "expected extension container cleared on nil slot")
}

func TestEnsureExtensionContainerIdempotent(t *testing.T) {
	t.Run("organism", func(t *testing.T) {
		var o Organism
		first := o.ensureExtensionContainer()
		if first == nil {
			t.Fatalf("expected container instance for zero-value organism")
		}
		if again := o.ensureExtensionContainer(); again != first {
			t.Fatalf("expected ensure to reuse container instance")
		}
		if l := len(first.Plugins(extension.HookOrganismAttributes)); l != 0 {
			t.Fatalf("expected no plugins for nil attributes, got %d", l)
		}
		mustNoError(t, "SetCoreAttributes", o.SetCoreAttributes(map[string]any{"flag": true}))
		second := o.ensureExtensionContainer()
		if second == nil || second == first {
			t.Fatalf("expected container to refresh after attribute update")
		}
		if payload, ok := second.Get(extension.HookOrganismAttributes, extension.PluginCore); !ok || payload.(map[string]any)["flag"] != true {
			t.Fatalf("expected refreshed container to reflect attributes")
		}
	})

	t.Run("facility", func(t *testing.T) {
		var f Facility
		base := f.ensureExtensionContainer()
		if base == nil {
			t.Fatalf("expected container instance for zero-value facility")
		}
		mustNoError(t, "ApplyEnvironmentBaselines", f.ApplyEnvironmentBaselines(map[string]any{"temp": 20}))
		refreshed := f.ensureExtensionContainer()
		if refreshed == nil || refreshed == base {
			t.Fatalf("expected refreshed container for facility")
		}
		if payload, ok := refreshed.Get(extension.HookFacilityEnvironmentBaselines, extension.PluginCore); !ok || payload.(map[string]any)["temp"] != 20 {
			t.Fatalf("expected facility container to mirror baselines")
		}
		if final := f.ensureExtensionContainer(); final != refreshed {
			t.Fatalf("expected idempotent ensure for facility")
		}
	})

	t.Run("breeding", func(t *testing.T) {
		var b BreedingUnit
		_ = b.ensureExtensionContainer()
		mustNoError(t, "ApplyPairingAttributes", b.ApplyPairingAttributes(map[string]any{"note": "x"}))
		refreshed := b.ensureExtensionContainer()
		if refreshed == nil {
			t.Fatalf("expected breeding container to initialise")
		}
		if payload, ok := refreshed.Get(extension.HookBreedingUnitPairingAttributes, extension.PluginCore); !ok || payload.(map[string]any)["note"] != "x" {
			t.Fatalf("expected breeding container to mirror pairing attributes")
		}
		if final := b.ensureExtensionContainer(); final != refreshed {
			t.Fatalf("expected idempotent ensure for breeding unit")
		}
	})

	t.Run("observation", func(t *testing.T) {
		var o Observation
		_ = o.ensureExtensionContainer()
		mustNoError(t, "ApplyObservationData", o.ApplyObservationData(map[string]any{"metric": 1}))
		refreshed := o.ensureExtensionContainer()
		if refreshed == nil {
			t.Fatalf("expected observation container to initialise")
		}
		if payload, ok := refreshed.Get(extension.HookObservationData, extension.PluginCore); !ok || payload.(map[string]any)["metric"] != 1 {
			t.Fatalf("expected observation container to reflect data map")
		}
		if final := o.ensureExtensionContainer(); final != refreshed {
			t.Fatalf("expected idempotent ensure for observation")
		}
	})

	t.Run("sample", func(t *testing.T) {
		var s Sample
		_ = s.ensureExtensionContainer()
		mustNoError(t, "ApplySampleAttributes", s.ApplySampleAttributes(map[string]any{"batch": "A"}))
		refreshed := s.ensureExtensionContainer()
		if refreshed == nil {
			t.Fatalf("expected sample container to initialise")
		}
		if payload, ok := refreshed.Get(extension.HookSampleAttributes, extension.PluginCore); !ok || payload.(map[string]any)["batch"] != "A" {
			t.Fatalf("expected sample container to mirror attributes")
		}
		if final := s.ensureExtensionContainer(); final != refreshed {
			t.Fatalf("expected idempotent ensure for sample")
		}
	})

	t.Run("supply", func(t *testing.T) {
		var s SupplyItem
		_ = s.ensureExtensionContainer()
		mustNoError(t, "ApplySupplyAttributes", s.ApplySupplyAttributes(map[string]any{"flag": true}))
		refreshed := s.ensureExtensionContainer()
		if refreshed == nil {
			t.Fatalf("expected supply container to initialise")
		}
		if payload, ok := refreshed.Get(extension.HookSupplyItemAttributes, extension.PluginCore); !ok || payload.(map[string]any)["flag"] != true {
			t.Fatalf("expected supply container to mirror attributes")
		}
		if final := s.ensureExtensionContainer(); final != refreshed {
			t.Fatalf("expected idempotent ensure for supply item")
		}
	})
}

func TestFacilitySlotLifecycle(t *testing.T) {
	var facility Facility
	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(map[string]any{"temp": "20C"}))

	first := facility.EnsureEnvironmentBaselinesSlot()
	if facility.environmentBaselinesSlot != first {
		t.Fatalf("expected facility slot cache to be populated")
	}
	if again := facility.EnsureEnvironmentBaselinesSlot(); again != first {
		t.Fatalf("expected facility ensure to reuse cached slot")
	}

	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(map[string]any{"temp": "21C"}))

	second := facility.EnsureEnvironmentBaselinesSlot()
	if first == second {
		t.Fatalf("expected facility ensure to rebuild slot after reset")
	}
	payload, _ := second.Get(extension.PluginCore)
	if payload.(map[string]any)["temp"] != "21C" {
		t.Fatalf("expected facility slot to mirror new map value")
	}

	external := extension.NewSlot(extension.HookFacilityEnvironmentBaselines)
	if err := external.Set(extension.PluginCore, map[string]any{"temp": "22C"}); err != nil {
		t.Fatalf("set external payload: %v", err)
	}
	if err := facility.SetEnvironmentBaselinesSlot(external); err != nil {
		t.Fatalf("SetEnvironmentBaselinesSlot: %v", err)
	}
	if facility.environmentBaselinesSlot == external {
		t.Fatalf("expected facility to clone incoming slot")
	}
	if facility.EnvironmentBaselines()["temp"] != "22C" {
		t.Fatalf("expected facility map to reflect slot payload")
	}
	if facility.extensions == nil {
		t.Fatalf("expected facility extension container to be initialised")
	}

	// Mutate external slot; cached clone should remain unchanged.
	if err := external.Set(extension.PluginCore, map[string]any{"temp": "mutated"}); err != nil {
		t.Fatalf("mutate external slot: %v", err)
	}
	payload, _ = facility.EnsureEnvironmentBaselinesSlot().Get(extension.PluginCore)
	if payload.(map[string]any)["temp"] != "22C" {
		t.Fatalf("expected cloned slot to remain unchanged after external mutation")
	}

	if err := facility.SetEnvironmentBaselinesSlot(nil); err != nil {
		t.Fatalf("SetEnvironmentBaselinesSlot nil: %v", err)
	}
	if facility.EnvironmentBaselines() != nil || facility.environmentBaselinesSlot != nil {
		t.Fatalf("expected facility slot and view cleared when nil provided")
	}
	assertContainerEmpty(t, facility.extensions, "expected facility extension container cleared on nil slot")
}

func TestBreedingUnitSlotLifecycle(t *testing.T) {
	var unit BreedingUnit
	mustNoError(t, "ApplyPairingAttributes", unit.ApplyPairingAttributes(map[string]any{"note": "init"}))

	first := unit.EnsurePairingAttributesSlot()
	if unit.pairingAttributesSlot != first {
		t.Fatalf("expected breeding unit slot cache to populate")
	}
	if again := unit.EnsurePairingAttributesSlot(); again != first {
		t.Fatalf("expected breeding unit ensure to reuse cached slot")
	}

	mustNoError(t, "ApplyPairingAttributes", unit.ApplyPairingAttributes(map[string]any{"note": "updated"}))

	second := unit.EnsurePairingAttributesSlot()
	if first == second {
		t.Fatalf("expected fresh slot after reset")
	}
	payload, _ := second.Get(extension.PluginCore)
	if payload.(map[string]any)["note"] != "updated" {
		t.Fatalf("expected slot payload to mirror updated map")
	}

	incoming := extension.NewSlot(extension.HookBreedingUnitPairingAttributes)
	if err := incoming.Set(extension.PluginCore, map[string]any{"note": "slot"}); err != nil {
		t.Fatalf("set incoming slot: %v", err)
	}
	if err := unit.SetPairingAttributesSlot(incoming); err != nil {
		t.Fatalf("SetPairingAttributesSlot: %v", err)
	}
	if unit.pairingAttributesSlot == incoming {
		t.Fatalf("expected breeding unit to clone incoming slot")
	}
	if unit.PairingAttributes()["note"] != "slot" {
		t.Fatalf("expected map to reflect cloned slot payload")
	}
	if unit.extensions == nil {
		t.Fatalf("expected breeding unit extension container to be initialised")
	}

	if err := unit.SetPairingAttributesSlot(nil); err != nil {
		t.Fatalf("SetPairingAttributesSlot nil: %v", err)
	}
	if unit.PairingAttributes() != nil || unit.pairingAttributesSlot != nil {
		t.Fatalf("expected breeding unit slot and view cleared when nil provided")
	}
	assertContainerEmpty(t, unit.extensions, "expected breeding unit extension container cleared on nil slot")
}

func TestObservationSlotLifecycle(t *testing.T) {
	var obs Observation
	mustNoError(t, "ApplyObservationData", obs.ApplyObservationData(map[string]any{"metric": 1}))

	first := obs.EnsureObservationDataSlot()
	if obs.dataSlot != first {
		t.Fatalf("expected observation slot cache to populate")
	}
	if again := obs.EnsureObservationDataSlot(); again != first {
		t.Fatalf("expected observation ensure to reuse cached slot")
	}

	mustNoError(t, "ApplyObservationData", obs.ApplyObservationData(map[string]any{"metric": 2}))

	second := obs.EnsureObservationDataSlot()
	if first == second {
		t.Fatalf("expected fresh slot after reset")
	}
	payload, _ := second.Get(extension.PluginCore)
	if payload.(map[string]any)["metric"] != 2 {
		t.Fatalf("expected slot payload to mirror updated map")
	}

	incoming := extension.NewSlot(extension.HookObservationData)
	if err := incoming.Set(extension.PluginCore, map[string]any{"metric": 3}); err != nil {
		t.Fatalf("set incoming slot: %v", err)
	}
	if err := obs.SetObservationDataSlot(incoming); err != nil {
		t.Fatalf("SetObservationDataSlot: %v", err)
	}
	if obs.dataSlot == incoming {
		t.Fatalf("expected observation to clone incoming slot")
	}
	if obs.ObservationData()["metric"] != 3 {
		t.Fatalf("expected map to reflect cloned slot payload")
	}
	if obs.extensions == nil {
		t.Fatalf("expected observation extension container to be initialised")
	}

	if err := obs.SetObservationDataSlot(nil); err != nil {
		t.Fatalf("SetObservationDataSlot nil: %v", err)
	}
	if obs.ObservationData() != nil || obs.dataSlot != nil {
		t.Fatalf("expected observation slot and view cleared when nil provided")
	}
	assertContainerEmpty(t, obs.extensions, "expected observation extension container cleared on nil slot")
}

func TestSampleSlotLifecycle(t *testing.T) {
	var sample Sample
	mustNoError(t, "ApplySampleAttributes", sample.ApplySampleAttributes(map[string]any{"batch": "A"}))

	first := sample.EnsureSampleAttributesSlot()
	if sample.attributesSlot != first {
		t.Fatalf("expected sample slot cache to populate")
	}
	if again := sample.EnsureSampleAttributesSlot(); again != first {
		t.Fatalf("expected sample ensure to reuse cached slot")
	}

	mustNoError(t, "ApplySampleAttributes", sample.ApplySampleAttributes(map[string]any{"batch": "B"}))

	second := sample.EnsureSampleAttributesSlot()
	if first == second {
		t.Fatalf("expected fresh sample slot after reset")
	}
	payload, _ := second.Get(extension.PluginCore)
	if payload.(map[string]any)["batch"] != "B" {
		t.Fatalf("expected slot payload to mirror updated map")
	}

	incoming := extension.NewSlot(extension.HookSampleAttributes)
	if err := incoming.Set(extension.PluginCore, map[string]any{"batch": "C"}); err != nil {
		t.Fatalf("set incoming slot: %v", err)
	}
	if err := sample.SetSampleAttributesSlot(incoming); err != nil {
		t.Fatalf("SetSampleAttributesSlot: %v", err)
	}
	if sample.attributesSlot == incoming {
		t.Fatalf("expected sample to clone incoming slot")
	}
	if sample.SampleAttributes()["batch"] != "C" {
		t.Fatalf("expected map to reflect cloned slot payload")
	}
	if sample.extensions == nil {
		t.Fatalf("expected sample extension container to be initialised")
	}

	if err := sample.SetSampleAttributesSlot(nil); err != nil {
		t.Fatalf("SetSampleAttributesSlot nil: %v", err)
	}
	if sample.SampleAttributes() != nil {
		t.Fatalf("expected sample slot view cleared when nil provided")
	}
	assertSlotEmpty(t, sample.attributesSlot, "expected sample slot cleared when nil provided")
	assertContainerEmpty(t, sample.extensions, "expected sample extension container cleared on nil slot")
}

func TestSupplyItemSlotLifecycle(t *testing.T) {
	var supply SupplyItem
	mustNoError(t, "ApplySupplyAttributes", supply.ApplySupplyAttributes(map[string]any{"flag": true}))

	first := supply.EnsureSupplyItemAttributesSlot()
	if supply.attributesSlot != first {
		t.Fatalf("expected supply slot cache to populate")
	}
	if again := supply.EnsureSupplyItemAttributesSlot(); again != first {
		t.Fatalf("expected supply ensure to reuse cached slot")
	}

	mustNoError(t, "ApplySupplyAttributes", supply.ApplySupplyAttributes(map[string]any{"flag": false}))

	second := supply.EnsureSupplyItemAttributesSlot()
	if first == second {
		t.Fatalf("expected fresh supply slot after reset")
	}
	payload, _ := second.Get(extension.PluginCore)
	if payload.(map[string]any)["flag"] != false {
		t.Fatalf("expected slot payload to mirror updated map")
	}

	incoming := extension.NewSlot(extension.HookSupplyItemAttributes)
	if err := incoming.Set(extension.PluginCore, map[string]any{"flag": "slot"}); err != nil {
		t.Fatalf("set incoming slot: %v", err)
	}
	if err := supply.SetSupplyItemAttributesSlot(incoming); err != nil {
		t.Fatalf("SetSupplyItemAttributesSlot: %v", err)
	}
	if supply.attributesSlot == incoming {
		t.Fatalf("expected supply to clone incoming slot")
	}
	if supply.SupplyAttributes()["flag"] != "slot" {
		t.Fatalf("expected map to reflect cloned slot payload")
	}
	if supply.extensions == nil {
		t.Fatalf("expected supply extension container to be initialised")
	}

	if err := supply.SetSupplyItemAttributesSlot(nil); err != nil {
		t.Fatalf("SetSupplyItemAttributesSlot nil: %v", err)
	}
	if supply.SupplyAttributes() != nil {
		t.Fatalf("expected supply slot view cleared when nil provided")
	}
	assertSlotEmpty(t, supply.attributesSlot, "expected supply slot cleared when nil provided")
	assertContainerEmpty(t, supply.extensions, "expected supply extension container cleared on nil slot")
}

func TestFacilityEnvironmentBaselinesHelpersClone(t *testing.T) {
	input := map[string]any{"temp": []int{20}}

	var facility Facility
	mustNoError(t, "ApplyEnvironmentBaselines", facility.ApplyEnvironmentBaselines(input))

	input["temp"].([]int)[0] = 99
	if facility.EnvironmentBaselines()["temp"].([]int)[0] != 20 {
		t.Fatalf("expected facility baselines to be cloned from input")
	}

	baselines := facility.EnvironmentBaselines()
	baselines["temp"].([]int)[0] = 30
	if facility.EnvironmentBaselines()["temp"].([]int)[0] != 20 {
		t.Fatalf("expected facility baselines to remain unchanged after view mutation")
	}
}

func TestSampleSetAttributesNil(t *testing.T) {
	var sample Sample
	mustNoError(t, "ApplySampleAttributes", sample.ApplySampleAttributes(nil))
	if sample.SampleAttributes() != nil {
		t.Fatalf("expected nil attributes when input is nil")
	}
	if sample.SampleAttributes() != nil {
		t.Fatalf("expected nil attribute map when stored map is nil")
	}
}

func TestOrganismAttributesSlotBridgeRoundTrip(t *testing.T) {
	organism := Organism{}
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"flag": true}))

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
	if organism.CoreAttributes()["flag"] != true {
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

	if value, ok := organism.CoreAttributes()["flag"].(bool); !ok || value {
		t.Fatalf("expected organism attributes to be updated from slot")
	}
}

func TestExtensionSlotBridgeRoundTripsOtherEntities(t *testing.T) {
	tempMap := map[string]any{"temp": "22C"}
	env := Facility{}
	mustNoError(t, "ApplyEnvironmentBaselines", env.ApplyEnvironmentBaselines(tempMap))
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
	if env.EnvironmentBaselines()["temp"] != "25C" {
		t.Fatalf("expected facility baselines to reflect slot payload")
	}

	breeding := BreedingUnit{}
	mustNoError(t, "ApplyPairingAttributes", breeding.ApplyPairingAttributes(map[string]any{"note": "initial"}))
	bSlot := breeding.EnsurePairingAttributesSlot()
	if bSlot.Hook() != extension.HookBreedingUnitPairingAttributes {
		t.Fatalf("expected breeding unit hook %q", extension.HookBreedingUnitPairingAttributes)
	}
	_ = bSlot.Set(extension.PluginCore, map[string]any{"note": "updated"})
	if err := breeding.SetPairingAttributesSlot(bSlot); err != nil {
		t.Fatalf("unexpected error applying breeding slot: %v", err)
	}
	if breeding.PairingAttributes()["note"] != "updated" {
		t.Fatalf("expected breeding attributes to reflect slot payload")
	}

	observation := Observation{}
	mustNoError(t, "ApplyObservationData", observation.ApplyObservationData(map[string]any{"metric": 3}))
	oSlot := observation.EnsureObservationDataSlot()
	if oSlot.Hook() != extension.HookObservationData {
		t.Fatalf("expected observation hook %q", extension.HookObservationData)
	}
	_ = oSlot.Set(extension.PluginCore, map[string]any{"metric": 4})
	if err := observation.SetObservationDataSlot(oSlot); err != nil {
		t.Fatalf("unexpected error applying observation slot: %v", err)
	}
	if observation.ObservationData()["metric"] != 4 {
		t.Fatalf("expected observation data to reflect slot payload")
	}

	sample := Sample{}
	mustNoError(t, "ApplySampleAttributes", sample.ApplySampleAttributes(map[string]any{"batch": "A"}))
	sSlot := sample.EnsureSampleAttributesSlot()
	if sSlot.Hook() != extension.HookSampleAttributes {
		t.Fatalf("expected sample hook %q", extension.HookSampleAttributes)
	}
	_ = sSlot.Set(extension.PluginCore, map[string]any{"batch": "B"})
	if err := sample.SetSampleAttributesSlot(sSlot); err != nil {
		t.Fatalf("unexpected error applying sample slot: %v", err)
	}
	if sample.SampleAttributes()["batch"] != "B" {
		t.Fatalf("expected sample attributes to reflect slot payload")
	}

	supply := SupplyItem{}
	mustNoError(t, "ApplySupplyAttributes", supply.ApplySupplyAttributes(map[string]any{"reorder": true}))
	uSlot := supply.EnsureSupplyItemAttributesSlot()
	if uSlot.Hook() != extension.HookSupplyItemAttributes {
		t.Fatalf("expected supply hook %q", extension.HookSupplyItemAttributes)
	}
	_ = uSlot.Set(extension.PluginCore, map[string]any{"reorder": false})
	if err := supply.SetSupplyItemAttributesSlot(uSlot); err != nil {
		t.Fatalf("unexpected error applying supply slot: %v", err)
	}
	if supply.SupplyAttributes()["reorder"] != false {
		t.Fatalf("expected supply attributes to reflect slot payload")
	}
}

func TestExtensionSlotBridgeRejectsNonCorePlugins(t *testing.T) {
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	if err := slot.Set(extension.PluginID("external.plugin"), map[string]any{"flag": true}); err != nil {
		t.Fatalf("expected slot set to succeed, got %v", err)
	}
	var organism Organism
	if err := organism.SetAttributesSlot(slot); err != nil {
		t.Fatalf("expected non-core payload to be accepted: %v", err)
	}
	if organism.CoreAttributes() != nil {
		t.Fatalf("expected organism attributes map to be nil when only external plugins provided")
	}
	payload := organism.EnsureAttributesSlot()
	if len(payload.Plugins()) != 1 || payload.Plugins()[0] != extension.PluginID("external.plugin") {
		t.Fatalf("expected external plugin payload to persist, got %+v", payload.Plugins())
	}
}

func TestExtensionSlotBridgeRejectsNonObjectPayload(t *testing.T) {
	slot := extension.NewSlot(extension.HookOrganismAttributes)
	if err := slot.Set(extension.PluginCore, "scalar"); err == nil {
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
	if sample.SampleAttributes() != nil {
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
	if err := line.SetDefaultAttributesSlot(defaultSlot); err != nil {
		t.Fatalf("SetDefaultAttributesSlot: %v", err)
	}
	if line.defaultAttributesSlot == nil {
		t.Fatalf("expected default attributes slot to be rebound")
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
	if err := line.SetExtensionOverridesSlot(overrideSlot); err != nil {
		t.Fatalf("SetExtensionOverridesSlot: %v", err)
	}
	if line.extensionOverridesSlot == nil {
		t.Fatalf("expected extension overrides slot to be rebound")
	}

	strain := &Strain{}
	strainSlot := strain.EnsureAttributes()
	if strainSlot.Hook() != extension.HookStrainAttributes {
		t.Fatalf("expected strain hook %q", extension.HookStrainAttributes)
	}
	if err := strainSlot.Set(extension.PluginCore, map[string]any{"note": "strain"}); err != nil {
		t.Fatalf("set strain slot: %v", err)
	}
	if err := strain.SetAttributesSlot(strainSlot); err != nil {
		t.Fatalf("SetAttributesSlot (strain): %v", err)
	}
	if strain.attributesSlot == nil {
		t.Fatalf("expected strain slot to be rebound after set")
	}

	genotype := &GenotypeMarker{}
	genotypeSlot := genotype.EnsureAttributes()
	if genotypeSlot.Hook() != extension.HookGenotypeMarkerAttributes {
		t.Fatalf("expected genotype marker hook %q", extension.HookGenotypeMarkerAttributes)
	}
	if err := genotypeSlot.Set(extension.PluginCore, map[string]any{"note": "geno"}); err != nil {
		t.Fatalf("set genotype slot: %v", err)
	}
	if err := genotype.SetAttributesSlot(genotypeSlot); err != nil {
		t.Fatalf("SetAttributesSlot (genotype): %v", err)
	}
	if genotype.attributesSlot == nil {
		t.Fatalf("expected genotype slot to be rebound after set")
	}

	// Simulate JSON unmarshalling that loses hook binding.
	unboundSlot := extension.NewSlot("")
	strain.attributesSlot = unboundSlot
	genotype.attributesSlot = unboundSlot.Clone()
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
