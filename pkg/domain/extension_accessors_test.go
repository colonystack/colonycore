package domain

import (
	"testing"
	"time"

	"colonycore/pkg/domain/extension"
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

	if err := organism.SetCoreAttributes(nil); err != nil {
		t.Fatalf("SetCoreAttributes nil: %v", err)
	}
	if attrs := organism.CoreAttributes(); attrs != nil {
		t.Fatalf("expected attributes cleared after nil assignment")
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

	if err := facility.SetFacilityExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetFacilityExtensions empty: %v", err)
	}
	if facility.EnvironmentBaselines() != nil {
		t.Fatalf("expected baselines cleared after empty container assignment")
	}
}

const testNotesAttribute = "updated"

func TestObservationAndSampleHooks(t *testing.T) {
	now := time.Now()

	observation := Observation{Base: Base{ID: "obs", CreatedAt: now, UpdatedAt: now}}
	if err := observation.ApplyObservationData(map[string]any{"notes": "value"}); err != nil {
		t.Fatalf("ApplyObservationData: %v", err)
	}
	if data := observation.ObservationData(); data["notes"] != "value" {
		t.Fatalf("unexpected observation data: %+v", data)
	}
	oext, err := observation.ObservationExtensions()
	if err != nil {
		t.Fatalf("ObservationExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&oext, extension.HookObservationData, extension.PluginCore); !ok || payload["notes"] != "value" {
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
	if err := observation.ApplyObservationData(nil); err != nil {
		t.Fatalf("ApplyObservationData nil: %v", err)
	}
	if observation.ObservationData() != nil {
		t.Fatalf("expected observation data cleared on nil")
	}

	sample := Sample{Base: Base{ID: "sample", CreatedAt: now, UpdatedAt: now}}
	if err := sample.ApplySampleAttributes(map[string]any{"volume": "5ml"}); err != nil {
		t.Fatalf("ApplySampleAttributes: %v", err)
	}
	if attrs := sample.SampleAttributes(); attrs["volume"] != "5ml" {
		t.Fatalf("unexpected sample attrs: %+v", attrs)
	}
	sext, err := sample.SampleExtensions()
	if err != nil {
		t.Fatalf("SampleExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&sext, extension.HookSampleAttributes, extension.PluginCore); !ok || payload["volume"] != "5ml" {
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
	if err := sample.ApplySampleAttributes(nil); err != nil {
		t.Fatalf("ApplySampleAttributes nil: %v", err)
	}
	if attrs := sample.SampleAttributes(); attrs != nil {
		t.Fatalf("expected sample attrs cleared on nil")
	}
}

func TestSupplyItemExtensionsLifecycle(t *testing.T) {
	var item SupplyItem
	if err := item.ApplySupplyAttributes(map[string]any{"vendor": "acme"}); err != nil {
		t.Fatalf("ApplySupplyAttributes: %v", err)
	}
	itemAttributes := item.SupplyAttributes()
	itemAttributes["vendor"] = "mutated"
	if refreshed := item.SupplyAttributes(); refreshed["vendor"] != "acme" {
		t.Fatalf("expected stored vendor to remain immutable: %+v", refreshed)
	}

	container := extension.NewContainer()
	if err := container.Set(extension.HookSupplyItemAttributes, extension.PluginCore, map[string]any{"vendor": "global"}); err != nil {
		t.Fatalf("set container: %v", err)
	}
	if err := item.SetSupplyItemExtensions(container); err != nil {
		t.Fatalf("SetSupplyItemExtensions: %v", err)
	}
	if attrs := item.SupplyAttributes(); attrs["vendor"] != "global" {
		t.Fatalf("expected container payload applied, got %+v", attrs)
	}
	sext, err := item.SupplyItemExtensions()
	if err != nil {
		t.Fatalf("SupplyItemExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&sext, extension.HookSupplyItemAttributes, extension.PluginCore); !ok || payload["vendor"] != "global" {
		t.Fatalf("unexpected supply extension payload: %+v", payload)
	}
	if err := item.SetSupplyItemExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetSupplyItemExtensions empty: %v", err)
	}
	if attrs := item.SupplyAttributes(); attrs != nil {
		t.Fatalf("expected supply attributes cleared after empty container assignment")
	}
}

func TestBreedingUnitExtensionsLifecycle(t *testing.T) {
	var unit BreedingUnit
	if err := unit.ApplyPairingAttributes(map[string]any{"strategy": "outcross"}); err != nil {
		t.Fatalf("ApplyPairingAttributes: %v", err)
	}
	bext, err := unit.BreedingUnitExtensions()
	if err != nil {
		t.Fatalf("BreedingUnitExtensions: %v", err)
	}
	if payload, ok := cloneHookMap(&bext, extension.HookBreedingUnitPairingAttributes, extension.PluginCore); !ok || payload["strategy"] != "outcross" {
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
	if err := unit.SetBreedingUnitExtensions(extension.NewContainer()); err != nil {
		t.Fatalf("SetBreedingUnitExtensions empty: %v", err)
	}
	if attrs := unit.PairingAttributes(); attrs != nil {
		t.Fatalf("expected attributes cleared after empty container assignment")
	}
}
