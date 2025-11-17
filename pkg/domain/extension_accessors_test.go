package domain

import (
	"testing"
	"time"

	"colonycore/pkg/domain/extension"
)

const (
	testTypedValue      = "typed"
	testSampleVolume    = "5ml"
	testSupplyVendor    = "global"
	testPairingStrategy = "outcross"
)

func TestCloneHookMapVariants(t *testing.T) {
	if _, ok := cloneHookMap(nil, extension.HookOrganismAttributes, extension.PluginCore); ok {
		t.Fatalf("expected no payload when container is nil")
	}

	container := extension.NewContainer()
	if _, ok := cloneHookMap(&container, extension.HookOrganismAttributes, extension.PluginCore); ok {
		t.Fatalf("expected no payload when hook is empty")
	}

	otherPayload := map[string]any{"flag": true}
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginID("external.plugin"), otherPayload); err != nil {
		t.Fatalf("set external payload: %v", err)
	}
	if _, ok := cloneHookMap(&container, extension.HookOrganismAttributes, extension.PluginCore); ok {
		t.Fatalf("expected no payload for non-core plugin")
	}

	payload := map[string]any{"values": []int{1, 2, 3}}
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginCore, payload); err != nil {
		t.Fatalf("set core payload: %v", err)
	}

	cloned, ok := cloneHookMap(&container, extension.HookOrganismAttributes, extension.PluginCore)
	if !ok {
		t.Fatalf("expected payload for core plugin")
	}
	clonedSlice := cloned["values"].([]int)
	clonedSlice[0] = 99

	again, _ := cloneHookMap(&container, extension.HookOrganismAttributes, extension.PluginCore)
	originalSlice := again["values"].([]int)
	if originalSlice[0] != 1 {
		t.Fatalf("expected deep clone to protect stored payload, got %d", originalSlice[0])
	}

}

func TestCloneContainerNilAndCopy(t *testing.T) {
	emptyClone, err := cloneContainer(nil)
	if err != nil {
		t.Fatalf("clone nil container: %v", err)
	}
	if len(emptyClone.Hooks()) != 0 {
		t.Fatalf("expected empty clone for nil container, got hooks: %v", emptyClone.Hooks())
	}

	source := extension.NewContainer()
	if err := source.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"flag": true}); err != nil {
		t.Fatalf("set source payload: %v", err)
	}
	clone, err := cloneContainer(&source)
	if err != nil {
		t.Fatalf("clone container: %v", err)
	}
	if len(clone.Hooks()) != 1 {
		t.Fatalf("expected single hook in clone, got %d", len(clone.Hooks()))
	}
	clonePayload, ok := cloneHookMap(&clone, extension.HookOrganismAttributes, extension.PluginCore)
	if !ok || clonePayload["flag"] != true {
		t.Fatalf("unexpected clone payload: %+v", clonePayload)
	}
	clonePayload["flag"] = false
	originalPayload, _ := cloneHookMap(&source, extension.HookOrganismAttributes, extension.PluginCore)
	if originalPayload["flag"] != true {
		t.Fatalf("expected original payload to remain unchanged")
	}
}

func TestOrganismCoreAttributesLifecycle(t *testing.T) {
	var organism Organism
	if attrs := organism.CoreAttributes(); attrs != nil {
		t.Fatalf("expected nil attributes for zero-value organism")
	}
	initialPayload := organism.CoreAttributesPayload()
	if !initialPayload.Defined() {
		t.Fatalf("expected payload to be initialised for hook")
	}
	if !initialPayload.IsEmpty() {
		t.Fatalf("expected zero-value payload to be empty")
	}

	input := map[string]any{"energy": 42}
	if err := organism.SetCoreAttributes(input); err != nil {
		t.Fatalf("SetCoreAttributes: %v", err)
	}
	input["energy"] = 21

	attrs := organism.CoreAttributes()
	if attrs["energy"] != 42 {
		t.Fatalf("expected cloned payload 42, got %v", attrs["energy"])
	}
	attrs["energy"] = 7
	if refreshed := organism.CoreAttributes(); refreshed["energy"] != 42 {
		t.Fatalf("expected stored payload to remain immutable, got %v", refreshed["energy"])
	}
	payload := organism.CoreAttributesPayload()
	if payload.Map()["energy"] != 42 {
		t.Fatalf("expected payload wrapper to match cloned value")
	}
	typedUpdate, err := extension.NewObjectPayload(extension.HookOrganismAttributes, map[string]any{"energy": 99})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	if err := organism.SetCoreAttributesPayload(typedUpdate); err != nil {
		t.Fatalf("SetCoreAttributesPayload: %v", err)
	}
	if got := organism.CoreAttributesPayload().Map()["energy"]; got != 99 {
		t.Fatalf("expected typed payload update to apply, got %v", got)
	}

	if err := organism.SetCoreAttributes(nil); err != nil {
		t.Fatalf("SetCoreAttributes nil: %v", err)
	}
	if attrs := organism.CoreAttributes(); attrs != nil {
		t.Fatalf("expected attributes cleared after nil assignment")
	}
	if cleared := organism.CoreAttributesPayload(); !cleared.IsEmpty() {
		t.Fatalf("expected payload wrapper to report empty after nil assignment")
	}
}

func TestOrganismExtensionsRoundTrip(t *testing.T) {
	var organism Organism

	container := extension.NewContainer()
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"flag": true}); err != nil {
		t.Fatalf("set container: %v", err)
	}
	if err := organism.SetOrganismExtensions(container); err != nil {
		t.Fatalf("SetOrganismExtensions: %v", err)
	}

	cloned, err := organism.OrganismExtensions()
	if err != nil {
		t.Fatalf("OrganismExtensions: %v", err)
	}
	payload, ok := cloneHookMap(&cloned, extension.HookOrganismAttributes, extension.PluginCore)
	if !ok || payload["flag"] != true {
		t.Fatalf("unexpected payload from clone: %+v", payload)
	}
	payload["flag"] = false
	current := organism.CoreAttributes()
	if current["flag"] != true {
		t.Fatalf("expected organism payload to remain unchanged")
	}

	if err := organism.SetOrganismExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetOrganismExtensions empty: %v", err)
	}
	if attrs := organism.CoreAttributes(); attrs != nil {
		t.Fatalf("expected attributes cleared after empty container assign")
	}
}

func TestSetCoreAttributesPayloadHookValidation(t *testing.T) {
	var organism Organism
	wrong, err := extension.NewObjectPayload(extension.HookFacilityEnvironmentBaselines, map[string]any{"temp": 20})
	if err != nil {
		t.Fatalf("NewObjectPayload wrong hook: %v", err)
	}
	if err := organism.SetCoreAttributesPayload(wrong); err == nil {
		t.Fatalf("expected hook mismatch to return error")
	}
	var zero extension.ObjectPayload
	if err := organism.SetCoreAttributesPayload(zero); err == nil {
		t.Fatalf("expected uninitialised payload to return error")
	}
}

func TestUpdateHookPayloadRetainsNonCorePlugins(t *testing.T) {
	var organism Organism
	container := extension.NewContainer()
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"flag": true}); err != nil {
		t.Fatalf("set core payload: %v", err)
	}
	if err := container.Set(extension.HookOrganismAttributes, extension.PluginID("external.plugin"), map[string]any{"note": "x"}); err != nil {
		t.Fatalf("set external payload: %v", err)
	}
	if err := organism.SetOrganismExtensions(container); err != nil {
		t.Fatalf("SetOrganismExtensions: %v", err)
	}
	if err := organism.SetCoreAttributes(nil); err != nil {
		t.Fatalf("SetCoreAttributes nil: %v", err)
	}
	if attrs := organism.CoreAttributes(); attrs != nil {
		t.Fatalf("expected core attributes cleared")
	}
	extensions, err := organism.OrganismExtensions()
	if err != nil {
		t.Fatalf("OrganismExtensions: %v", err)
	}
	plugins := extensions.Plugins(extension.HookOrganismAttributes)
	if len(plugins) != 1 || plugins[0] != extension.PluginID("external.plugin") {
		t.Fatalf("expected external plugin to remain, got %v", plugins)
	}
	payload, ok := cloneHookMap(&extensions, extension.HookOrganismAttributes, extension.PluginID("external.plugin"))
	if !ok || payload["note"] != "x" {
		t.Fatalf("expected external payload preserved, got %+v", payload)
	}
}

func TestFacilityEnvironmentBaselinesLifecycle(t *testing.T) {
	var facility Facility
	src := map[string]any{"temp": []int{20, 22}}
	if err := facility.ApplyEnvironmentBaselines(src); err != nil {
		t.Fatalf("ApplyEnvironmentBaselines: %v", err)
	}
	src["temp"].([]int)[0] = 99

	baselines := facility.EnvironmentBaselines()
	if got := baselines["temp"].([]int)[0]; got != 20 {
		t.Fatalf("expected cloned baseline 20, got %d", got)
	}
	baselines["temp"].([]int)[1] = 30
	if again := facility.EnvironmentBaselines()["temp"].([]int)[1]; again != 22 {
		t.Fatalf("expected stored baseline to remain immutable, got %d", again)
	}
	payload := facility.EnvironmentBaselinesPayload()
	if !payload.Defined() {
		t.Fatalf("expected payload wrapper defined for facility hook")
	}
	if payload.Map()["temp"].([]int)[1] != 22 {
		t.Fatalf("expected payload wrapper to match stored value")
	}

	ext, err := facility.FacilityExtensions()
	if err != nil {
		t.Fatalf("FacilityExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&ext, extension.HookFacilityEnvironmentBaselines, extension.PluginCore); !ok || payload["temp"].([]int)[0] != 20 {
		t.Fatalf("unexpected facility extension payload: %+v", payload)
	}

	updateContainer := extension.NewContainer()
	if err := updateContainer.Set(extension.HookFacilityEnvironmentBaselines, extension.PluginCore, map[string]any{"temp": []int{18}}); err != nil {
		t.Fatalf("set update container: %v", err)
	}
	if err := facility.SetFacilityExtensions(updateContainer); err != nil {
		t.Fatalf("SetFacilityExtensions update: %v", err)
	}
	if facility.EnvironmentBaselines()["temp"].([]int)[0] != 18 {
		t.Fatalf("expected updated baseline applied")
	}
	if facility.EnvironmentBaselinesPayload().Map()["temp"].([]int)[0] != 18 {
		t.Fatalf("expected payload wrapper to reflect updated baseline")
	}
	typed, err := extension.NewObjectPayload(extension.HookFacilityEnvironmentBaselines, map[string]any{"temp": []int{15}})
	if err != nil {
		t.Fatalf("NewObjectPayload facility: %v", err)
	}
	if err := facility.ApplyEnvironmentBaselinesPayload(typed); err != nil {
		t.Fatalf("ApplyEnvironmentBaselinesPayload: %v", err)
	}
	if facility.EnvironmentBaselines()["temp"].([]int)[0] != 15 {
		t.Fatalf("expected typed payload update applied")
	}

	if err := facility.SetFacilityExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetFacilityExtensions empty: %v", err)
	}
	if facility.EnvironmentBaselines() != nil {
		t.Fatalf("expected baselines cleared after empty container assignment")
	}
	if !facility.EnvironmentBaselinesPayload().IsEmpty() {
		t.Fatalf("expected payload wrapper to report empty after clear")
	}
}

const testNotesAttribute = "updated"

func TestObservationAndSampleHooks(t *testing.T) {
	now := time.Now()

	observation := Observation{Base: Base{ID: "obs", CreatedAt: now, UpdatedAt: now}}
	if err := observation.ApplyObservationData(map[string]any{"notes": testAttrValue}); err != nil {
		t.Fatalf("ApplyObservationData: %v", err)
	}
	if data := observation.ObservationData(); data["notes"] != testAttrValue {
		t.Fatalf("unexpected observation data: %+v", data)
	}
	dataPayload := observation.ObservationDataPayload()
	if !dataPayload.Defined() {
		t.Fatalf("expected observation payload defined")
	}
	if dataPayload.Map()["notes"] != testAttrValue {
		t.Fatalf("expected observation payload wrapper to match data")
	}
	oext, err := observation.ObservationExtensions()
	if err != nil {
		t.Fatalf("ObservationExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&oext, extension.HookObservationData, extension.PluginCore); !ok || payload["notes"] != testAttrValue {
		t.Fatalf("unexpected observation container payload: %+v", payload)
	}
	container := extension.NewContainer()
	if err := container.Set(extension.HookObservationData, extension.PluginCore, map[string]any{"notes": testNotesAttribute}); err != nil {
		t.Fatalf("set observation container: %v", err)
	}
	if err := observation.SetObservationExtensions(container); err != nil {
		t.Fatalf("SetObservationExtensions: %v", err)
	}
	if observation.ObservationData()["notes"] != testNotesAttribute {
		t.Fatalf("expected observation data replaced by container payload")
	}
	if observation.ObservationDataPayload().Map()["notes"] != testNotesAttribute {
		t.Fatalf("expected payload wrapper to reflect container data")
	}
	typedObs, err := extension.NewObjectPayload(extension.HookObservationData, map[string]any{"notes": testTypedValue})
	if err != nil {
		t.Fatalf("NewObjectPayload observation: %v", err)
	}
	if err := observation.ApplyObservationDataPayload(typedObs); err != nil {
		t.Fatalf("ApplyObservationDataPayload: %v", err)
	}
	if observation.ObservationData()["notes"] != testTypedValue {
		t.Fatalf("expected typed observation payload applied")
	}
	if err := observation.ApplyObservationData(nil); err != nil {
		t.Fatalf("ApplyObservationData nil: %v", err)
	}
	if observation.ObservationData() != nil {
		t.Fatalf("expected observation data cleared on nil")
	}
	if !observation.ObservationDataPayload().IsEmpty() {
		t.Fatalf("expected observation payload wrapper cleared")
	}

	sample := Sample{Base: Base{ID: "sample", CreatedAt: now, UpdatedAt: now}}
	if err := sample.ApplySampleAttributes(map[string]any{"volume": testSampleVolume}); err != nil {
		t.Fatalf("ApplySampleAttributes: %v", err)
	}
	if attrs := sample.SampleAttributes(); attrs["volume"] != testSampleVolume {
		t.Fatalf("unexpected sample attrs: %+v", attrs)
	}
	samplePayload := sample.SampleAttributesPayload()
	if !samplePayload.Defined() {
		t.Fatalf("expected sample payload defined")
	}
	if samplePayload.Map()["volume"] != testSampleVolume {
		t.Fatalf("expected sample payload wrapper to match attributes")
	}
	sext, err := sample.SampleExtensions()
	if err != nil {
		t.Fatalf("SampleExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&sext, extension.HookSampleAttributes, extension.PluginCore); !ok || payload["volume"] != testSampleVolume {
		t.Fatalf("unexpected sample extension payload: %+v", payload)
	}
	container = extension.NewContainer()
	if err := container.Set(extension.HookSampleAttributes, extension.PluginCore, map[string]any{"volume": "10ml"}); err != nil {
		t.Fatalf("set sample container: %v", err)
	}
	if err := sample.SetSampleExtensions(container); err != nil {
		t.Fatalf("SetSampleExtensions: %v", err)
	}
	if sample.SampleAttributes()["volume"] != "10ml" {
		t.Fatalf("expected sample attributes replaced by container payload")
	}
	if sample.SampleAttributesPayload().Map()["volume"] != "10ml" {
		t.Fatalf("expected sample payload wrapper to reflect container data")
	}
	typedSample, err := extension.NewObjectPayload(extension.HookSampleAttributes, map[string]any{"volume": testTypedValue})
	if err != nil {
		t.Fatalf("NewObjectPayload sample: %v", err)
	}
	if err := sample.ApplySampleAttributesPayload(typedSample); err != nil {
		t.Fatalf("ApplySampleAttributesPayload: %v", err)
	}
	if sample.SampleAttributes()["volume"] != testTypedValue {
		t.Fatalf("expected typed sample payload applied")
	}
	if err := sample.ApplySampleAttributes(nil); err != nil {
		t.Fatalf("ApplySampleAttributes nil: %v", err)
	}
	if attrs := sample.SampleAttributes(); attrs != nil {
		t.Fatalf("expected sample attrs cleared on nil")
	}
	if !sample.SampleAttributesPayload().IsEmpty() {
		t.Fatalf("expected sample payload wrapper cleared")
	}
}

func TestSupplyItemExtensionsLifecycle(t *testing.T) {
	var item SupplyItem
	if err := item.ApplySupplyAttributes(map[string]any{"vendor": "acme"}); err != nil {
		t.Fatalf("ApplySupplyAttributes: %v", err)
	}
	supplyPayload := item.SupplyAttributesPayload()
	if !supplyPayload.Defined() {
		t.Fatalf("expected supply payload defined")
	}
	if supplyPayload.Map()["vendor"] != "acme" {
		t.Fatalf("expected payload wrapper to match attributes")
	}
	itemAttributes := item.SupplyAttributes()
	itemAttributes["vendor"] = testMutated
	if refreshed := item.SupplyAttributes(); refreshed["vendor"] != "acme" {
		t.Fatalf("expected stored vendor to remain immutable: %+v", refreshed)
	}

	container := extension.NewContainer()
	if err := container.Set(extension.HookSupplyItemAttributes, extension.PluginCore, map[string]any{"vendor": testSupplyVendor}); err != nil {
		t.Fatalf("set container: %v", err)
	}
	if err := item.SetSupplyItemExtensions(container); err != nil {
		t.Fatalf("SetSupplyItemExtensions: %v", err)
	}
	if attrs := item.SupplyAttributes(); attrs["vendor"] != testSupplyVendor {
		t.Fatalf("expected container payload applied, got %+v", attrs)
	}
	sext, err := item.SupplyItemExtensions()
	if err != nil {
		t.Fatalf("SupplyItemExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&sext, extension.HookSupplyItemAttributes, extension.PluginCore); !ok || payload["vendor"] != testSupplyVendor {
		t.Fatalf("unexpected supply extension payload: %+v", payload)
	}
	if item.SupplyAttributesPayload().Map()["vendor"] != testSupplyVendor {
		t.Fatalf("expected payload wrapper to reflect container data")
	}
	typedSupply, err := extension.NewObjectPayload(extension.HookSupplyItemAttributes, map[string]any{"vendor": testTypedValue})
	if err != nil {
		t.Fatalf("NewObjectPayload supply: %v", err)
	}
	if err := item.ApplySupplyAttributesPayload(typedSupply); err != nil {
		t.Fatalf("ApplySupplyAttributesPayload: %v", err)
	}
	if item.SupplyAttributes()["vendor"] != testTypedValue {
		t.Fatalf("expected typed supply payload applied")
	}
	if err := item.SetSupplyItemExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetSupplyItemExtensions empty: %v", err)
	}
	if attrs := item.SupplyAttributes(); attrs != nil {
		t.Fatalf("expected supply attributes cleared after empty container assignment")
	}
	if !item.SupplyAttributesPayload().IsEmpty() {
		t.Fatalf("expected supply payload wrapper cleared")
	}
}

func TestBreedingUnitExtensionsLifecycle(t *testing.T) {
	var unit BreedingUnit
	if err := unit.ApplyPairingAttributes(map[string]any{"strategy": testPairingStrategy}); err != nil {
		t.Fatalf("ApplyPairingAttributes: %v", err)
	}
	pairPayload := unit.PairingAttributesPayload()
	if !pairPayload.Defined() {
		t.Fatalf("expected pairing payload defined")
	}
	if pairPayload.Map()["strategy"] != testPairingStrategy {
		t.Fatalf("expected pairing payload wrapper to match data")
	}
	bext, err := unit.BreedingUnitExtensions()
	if err != nil {
		t.Fatalf("BreedingUnitExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&bext, extension.HookBreedingUnitPairingAttributes, extension.PluginCore); !ok || payload["strategy"] != testPairingStrategy {
		t.Fatalf("unexpected breeding payload: %+v", payload)
	}
	container := extension.NewContainer()
	if err := container.Set(extension.HookBreedingUnitPairingAttributes, extension.PluginCore, map[string]any{"strategy": "inbred"}); err != nil {
		t.Fatalf("set container: %v", err)
	}
	if err := unit.SetBreedingUnitExtensions(container); err != nil {
		t.Fatalf("SetBreedingUnitExtensions: %v", err)
	}
	if attrs := unit.PairingAttributes(); attrs["strategy"] != "inbred" {
		t.Fatalf("expected pairing attributes replaced, got %+v", attrs)
	}
	if unit.PairingAttributesPayload().Map()["strategy"] != "inbred" {
		t.Fatalf("expected payload wrapper to reflect container payload")
	}
	typedPairing, err := extension.NewObjectPayload(extension.HookBreedingUnitPairingAttributes, map[string]any{"strategy": testTypedValue})
	if err != nil {
		t.Fatalf("NewObjectPayload pairing: %v", err)
	}
	if err := unit.ApplyPairingAttributesPayload(typedPairing); err != nil {
		t.Fatalf("ApplyPairingAttributesPayload: %v", err)
	}
	if unit.PairingAttributes()["strategy"] != testTypedValue {
		t.Fatalf("expected typed pairing payload applied")
	}
	if err := unit.SetBreedingUnitExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetBreedingUnitExtensions empty: %v", err)
	}
	if attrs := unit.PairingAttributes(); attrs != nil {
		t.Fatalf("expected attributes cleared after empty container assignment")
	}
	if !unit.PairingAttributesPayload().IsEmpty() {
		t.Fatalf("expected pairing payload wrapper cleared")
	}
}

func TestLineExtensionsRoundTrip(t *testing.T) {
	var line Line
	container := extension.NewContainer()
	if err := container.Set(extension.HookLineDefaultAttributes, extension.PluginCore, map[string]any{"seed": true}); err != nil {
		t.Fatalf("set default attributes: %v", err)
	}
	if err := container.Set(extension.HookLineExtensionOverrides, extension.PluginID("external"), map[string]any{"override": 1}); err != nil {
		t.Fatalf("set overrides: %v", err)
	}
	if err := line.SetLineExtensions(container); err != nil {
		t.Fatalf("SetLineExtensions: %v", err)
	}

	cloned, err := line.LineExtensions()
	if err != nil {
		t.Fatalf("LineExtensions: %v", err)
	}
	defaultPayload, ok := cloneHookMap(&cloned, extension.HookLineDefaultAttributes, extension.PluginCore)
	if !ok || defaultPayload["seed"] != true {
		t.Fatalf("unexpected default payload %+v", defaultPayload)
	}
	defaultPayload["seed"] = false
	slot := line.EnsureDefaultAttributes()
	if stored, ok := slot.Get(extension.PluginCore); !ok || stored.(map[string]any)["seed"] != true {
		t.Fatalf("expected stored default payload unchanged, got %+v", stored)
	}

	overrideSlot := line.EnsureExtensionOverrides()
	overridePayload, ok := overrideSlot.Get(extension.PluginID("external"))
	if !ok || overridePayload.(map[string]any)["override"] != 1 {
		t.Fatalf("unexpected override payload %+v", overridePayload)
	}

	if err := line.SetLineExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetLineExtensions empty: %v", err)
	}
	if line.defaultAttributesSlot != nil || line.extensionOverridesSlot != nil {
		t.Fatalf("expected line slots cleared after empty assignment")
	}

	bad := extension.NewContainer()
	if err := bad.Set(extension.HookOrganismAttributes, extension.PluginCore, map[string]any{"oops": true}); err != nil {
		t.Fatalf("set bad container: %v", err)
	}
	if err := line.SetLineExtensions(bad); err == nil {
		t.Fatalf("expected error for unsupported hook assignment")
	}
}

func TestLineDefaultAttributesAccessors(t *testing.T) {
	var line Line
	if attrs := line.DefaultAttributes(); attrs != nil {
		t.Fatalf("expected nil default attributes when unset")
	}

	input := map[string]any{
		extension.PluginCore.String(): map[string]any{"seed": "alpha"},
		"plugin.beta":                 map[string]any{"flag": true},
	}
	if err := line.ApplyDefaultAttributes(input); err != nil {
		t.Fatalf("ApplyDefaultAttributes: %v", err)
	}
	attrs := line.DefaultAttributes()
	if len(attrs) != 2 {
		t.Fatalf("expected two plugin payloads, got %d", len(attrs))
	}
	if attrs[extension.PluginCore.String()].(map[string]any)["seed"] != "alpha" {
		t.Fatalf("unexpected default payload: %+v", attrs)
	}
	input[extension.PluginCore.String()].(map[string]any)["seed"] = "mutated"
	reloaded := line.DefaultAttributes()
	if reloaded[extension.PluginCore.String()].(map[string]any)["seed"] != "alpha" {
		t.Fatalf("expected stored payload immune to caller mutation, got %+v", reloaded)
	}

	attrs["plugin.beta"].(map[string]any)["flag"] = false
	verify := line.DefaultAttributes()
	if verify["plugin.beta"].(map[string]any)["flag"] != true {
		t.Fatalf("expected cloned payload from accessor, got %+v", verify)
	}

	if err := line.ApplyDefaultAttributes(map[string]any{}); err != nil {
		t.Fatalf("ApplyDefaultAttributes empty: %v", err)
	}
	if attrs := line.DefaultAttributes(); attrs != nil {
		t.Fatalf("expected default attributes cleared after empty assignment")
	}
	if slot := line.defaultAttributesSlot; slot != nil && len(slot.Plugins()) != 0 {
		t.Fatalf("expected default attributes slot empty, got plugins %v", slot.Plugins())
	}
}

func TestLineExtensionOverrideAccessors(t *testing.T) {
	var line Line
	if overrides := line.ExtensionOverrides(); overrides != nil {
		t.Fatalf("expected nil overrides when unset")
	}

	const testPluginAttribute = "strict"
	overrideInput := map[string]any{
		extension.PluginCore.String(): map[string]any{"threshold": 1},
		"plugin.gamma":                map[string]any{"mode": testPluginAttribute},
	}
	if err := line.ApplyExtensionOverrides(overrideInput); err != nil {
		t.Fatalf("ApplyExtensionOverrides: %v", err)
	}
	overrides := line.ExtensionOverrides()
	if len(overrides) != 2 {
		t.Fatalf("expected overrides for two plugins, got %d", len(overrides))
	}
	if overrides["plugin.gamma"].(map[string]any)["mode"] != testPluginAttribute {
		t.Fatalf("unexpected overrides payload: %+v", overrides)
	}

	overrideInput["plugin.gamma"].(map[string]any)["mode"] = "relaxed"
	current := line.ExtensionOverrides()
	if current["plugin.gamma"].(map[string]any)["mode"] != testPluginAttribute {
		t.Fatalf("expected stored overrides immune to caller mutation, got %+v", current)
	}

	overrides["plugin.gamma"].(map[string]any)["mode"] = "debug"
	rebuilt := line.ExtensionOverrides()
	if rebuilt["plugin.gamma"].(map[string]any)["mode"] != testPluginAttribute {
		t.Fatalf("expected accessor to return cloned payload, got %+v", rebuilt)
	}

	if err := line.ApplyExtensionOverrides(nil); err != nil {
		t.Fatalf("ApplyExtensionOverrides nil: %v", err)
	}
	if line.ExtensionOverrides() != nil {
		t.Fatalf("expected overrides cleared after nil assignment")
	}
	if slot := line.extensionOverridesSlot; slot != nil && len(slot.Plugins()) != 0 {
		t.Fatalf("expected overrides slot empty, got plugins %v", slot.Plugins())
	}
}
func TestStrainExtensionsRoundTrip(t *testing.T) {
	var strain Strain
	container := extension.NewContainer()
	if err := container.Set(extension.HookStrainAttributes, extension.PluginCore, map[string]any{"note": "strain"}); err != nil {
		t.Fatalf("set strain container: %v", err)
	}
	if err := strain.SetStrainExtensions(container); err != nil {
		t.Fatalf("SetStrainExtensions: %v", err)
	}
	cloned, err := strain.StrainExtensions()
	if err != nil {
		t.Fatalf("StrainExtensions: %v", err)
	}
	payload, ok := cloneHookMap(&cloned, extension.HookStrainAttributes, extension.PluginCore)
	if !ok || payload["note"] != "strain" {
		t.Fatalf("unexpected strain payload %+v", payload)
	}
	payload["note"] = testMutated
	if stored, ok := strain.attributesSlot.Get(extension.PluginCore); !ok || stored.(map[string]any)["note"] != "strain" {
		t.Fatalf("expected stored strain payload unchanged")
	}

	if err := strain.SetStrainExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetStrainExtensions empty: %v", err)
	}
	if strain.attributesSlot != nil || strain.extensions != nil {
		t.Fatalf("expected strain slots cleared after empty assignment")
	}
}

func TestGenotypeMarkerExtensionsRoundTrip(t *testing.T) {
	var marker GenotypeMarker
	container := extension.NewContainer()
	if err := container.Set(extension.HookGenotypeMarkerAttributes, extension.PluginCore, map[string]any{"note": "marker"}); err != nil {
		t.Fatalf("set genotype container: %v", err)
	}
	if err := marker.SetGenotypeMarkerExtensions(container); err != nil {
		t.Fatalf("SetGenotypeMarkerExtensions: %v", err)
	}
	cloned, err := marker.GenotypeMarkerExtensions()
	if err != nil {
		t.Fatalf("GenotypeMarkerExtensions: %v", err)
	}
	payload, ok := cloneHookMap(&cloned, extension.HookGenotypeMarkerAttributes, extension.PluginCore)
	if !ok || payload["note"] != "marker" {
		t.Fatalf("unexpected genotype payload %+v", payload)
	}
	payload["note"] = testMutated
	if stored, ok := marker.attributesSlot.Get(extension.PluginCore); !ok || stored.(map[string]any)["note"] != "marker" {
		t.Fatalf("expected stored genotype payload unchanged")
	}

	if err := marker.SetGenotypeMarkerExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetGenotypeMarkerExtensions empty: %v", err)
	}
	if marker.attributesSlot != nil || marker.extensions != nil {
		t.Fatalf("expected genotype slots cleared after empty assignment")
	}
}

func TestAccessorPayloadsRejectMismatchedHooks(t *testing.T) {
	wrongOrganism, err := extension.NewObjectPayload(extension.HookFacilityEnvironmentBaselines, map[string]any{"temp": 20})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	var organism Organism
	if err := organism.SetCoreAttributesPayload(wrongOrganism); err == nil {
		t.Fatalf("expected hook mismatch for organism payload")
	}

	wrongFacility, err := extension.NewObjectPayload(extension.HookOrganismAttributes, map[string]any{"note": "bad"})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	var facility Facility
	if err := facility.ApplyEnvironmentBaselinesPayload(wrongFacility); err == nil {
		t.Fatalf("expected hook mismatch for facility payload")
	}

	wrongBreeding, err := extension.NewObjectPayload(extension.HookOrganismAttributes, map[string]any{"flag": true})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	var breeding BreedingUnit
	if err := breeding.ApplyPairingAttributesPayload(wrongBreeding); err == nil {
		t.Fatalf("expected hook mismatch for breeding payload")
	}

	wrongObservation, err := extension.NewObjectPayload(extension.HookSampleAttributes, map[string]any{"value": 1})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	var obs Observation
	if err := obs.ApplyObservationDataPayload(wrongObservation); err == nil {
		t.Fatalf("expected hook mismatch for observation payload")
	}

	wrongSample, err := extension.NewObjectPayload(extension.HookObservationData, map[string]any{"value": 1})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	var sample Sample
	if err := sample.ApplySampleAttributesPayload(wrongSample); err == nil {
		t.Fatalf("expected hook mismatch for sample payload")
	}

	wrongSupply, err := extension.NewObjectPayload(extension.HookObservationData, map[string]any{"value": 1})
	if err != nil {
		t.Fatalf("NewObjectPayload: %v", err)
	}
	var supply SupplyItem
	if err := supply.ApplySupplyAttributesPayload(wrongSupply); err == nil {
		t.Fatalf("expected hook mismatch for supply payload")
	}
}

func TestLineApplyHelpersRejectInvalidPlugins(t *testing.T) {
	var line Line
	if err := line.ApplyDefaultAttributes(map[string]any{"": map[string]any{}}); err == nil {
		t.Fatalf("expected default attributes error")
	}
	if err := line.ApplyExtensionOverrides(map[string]any{"": map[string]any{}}); err == nil {
		t.Fatalf("expected extension overrides error")
	}
}
