package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

func TestSlotFromMapClonesInputPayload(t *testing.T) {
	input := map[string]any{
		"outer": map[string]any{
			"inner": []any{1, map[string]any{"flag": true}},
		},
	}

	slot := slotFromMap(extension.HookOrganismAttributes, input)
	if slot.Hook() != extension.HookOrganismAttributes {
		t.Fatalf("expected hook %q, got %q", extension.HookOrganismAttributes, slot.Hook())
	}
	plugins := slot.Plugins()
	if len(plugins) != 1 || plugins[0] != extension.PluginCore {
		t.Fatalf("expected only core plugin, got %+v", plugins)
	}

	// Mutate original input; stored payload should remain unchanged.
	input["outer"].(map[string]any)["inner"].([]any)[0] = 99

	payloadRaw, ok := slot.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected payload for core plugin")
	}
	payload, ok := payloadRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected payload map, got %T", payloadRaw)
	}
	if got := payload["outer"].(map[string]any)["inner"].([]any)[0]; got != 1 {
		t.Fatalf("expected cloned payload to retain original value, got %v", got)
	}

	// Mutate payload clone; slot should remain immutable.
	payload["outer"].(map[string]any)["inner"].([]any)[0] = 55
	after, _ := slot.Get(extension.PluginCore)
	if got := after.(map[string]any)["outer"].(map[string]any)["inner"].([]any)[0]; got != 1 {
		t.Fatalf("expected slot payload to remain unchanged, got %v", got)
	}
}

func TestSlotFromMapWithNilInputProducesEmptySlot(t *testing.T) {
	slot := slotFromMap(extension.HookSampleAttributes, nil)
	if slot.Hook() != extension.HookSampleAttributes {
		t.Fatalf("expected hook %q, got %q", extension.HookSampleAttributes, slot.Hook())
	}
	if len(slot.Plugins()) != 0 {
		t.Fatalf("expected no plugins for nil input, got %+v", slot.Plugins())
	}
}

func TestMapFromSlotRoundTripClonesPayload(t *testing.T) {
	slot := extension.NewSlot(extension.HookSampleAttributes)
	if err := slot.Set(extension.PluginCore, map[string]any{"batch": "A"}); err != nil {
		t.Fatalf("set payload: %v", err)
	}

	result, err := mapFromSlot(extension.HookSampleAttributes, slot)
	if err != nil {
		t.Fatalf("mapFromSlot: %v", err)
	}
	if result["batch"] != "A" {
		t.Fatalf("expected batch 'A', got %v", result["batch"])
	}

	// Mutate cloned result and ensure slot payload stays intact.
	result["batch"] = "mutated"
	if payload, _ := slot.Get(extension.PluginCore); payload.(map[string]any)["batch"] != "A" {
		t.Fatalf("expected slot payload to remain unchanged")
	}
}

func TestMapFromSlotNilSlot(t *testing.T) {
	result, err := mapFromSlot(extension.HookOrganismAttributes, nil)
	if err != nil {
		t.Fatalf("mapFromSlot nil slot: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for nil slot, got %+v", result)
	}
}

func TestMapFromSlotRebindsHook(t *testing.T) {
	slot := extension.NewSlot(extension.HookSampleAttributes)
	if err := slot.Set(extension.PluginCore, map[string]any{"flag": true}); err != nil {
		t.Fatalf("set payload: %v", err)
	}

	result, err := mapFromSlot(extension.HookOrganismAttributes, slot)
	if err != nil {
		t.Fatalf("mapFromSlot with rebind: %v", err)
	}
	if slot.Hook() != extension.HookOrganismAttributes {
		t.Fatalf("expected slot hook to rebind to %q, got %q", extension.HookOrganismAttributes, slot.Hook())
	}
	if !result["flag"].(bool) {
		t.Fatalf("expected payload to survive rebind")
	}
}

func TestSlotFromContainerRoundTrip(t *testing.T) {
	var organism Organism
	organism.SetAttributes(map[string]any{"flag": true})

	slot := slotFromContainer(extension.HookOrganismAttributes, organism.ensureExtensionContainer())
	if slot == nil {
		t.Fatalf("expected slot from container")
	}
	if slot.Hook() != extension.HookOrganismAttributes {
		t.Fatalf("expected hook to remain %q", extension.HookOrganismAttributes)
	}
	payload, ok := slot.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected plugin payload")
	}
	if payload.(map[string]any)["flag"] != true {
		t.Fatalf("expected payload derived from container")
	}

	container, err := containerFromSlot(extension.HookOrganismAttributes, slot)
	if err != nil {
		t.Fatalf("containerFromSlot: %v", err)
	}
	if container == nil {
		t.Fatalf("expected container to be recreated from slot")
	}
	cloned, ok := container.Get(extension.HookOrganismAttributes, extension.PluginCore)
	if !ok {
		t.Fatalf("expected plugin payload in rebuilt container")
	}
	if cloned.(map[string]any)["flag"] != true {
		t.Fatalf("expected rebuilt container payload to match original")
	}
}

func TestSlotFromContainerNil(t *testing.T) {
	slot := slotFromContainer(extension.HookSampleAttributes, nil)
	if slot == nil {
		t.Fatalf("expected slot even when container nil")
	}
	if len(slot.Plugins()) != 0 {
		t.Fatalf("expected no plugins for nil container")
	}
}

func TestContainerFromSlotEmpty(t *testing.T) {
	slot := extension.NewSlot(extension.HookSampleAttributes)
	container, err := containerFromSlot(extension.HookSampleAttributes, slot)
	if err != nil {
		t.Fatalf("containerFromSlot empty: %v", err)
	}
	if container != nil {
		t.Fatalf("expected empty slot to produce nil container")
	}
}

func TestContainerFromSlotNil(t *testing.T) {
	container, err := containerFromSlot(extension.HookSampleAttributes, nil)
	if err != nil {
		t.Fatalf("containerFromSlot nil: %v", err)
	}
	if container != nil {
		t.Fatalf("expected nil slot to yield nil container")
	}
}

func TestMapFromSlotRejectsNonCorePlugin(t *testing.T) {
	slot := extension.NewSlot(extension.HookSampleAttributes)
	if err := slot.Set(extension.PluginID("external.plugin"), map[string]any{"field": true}); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if _, err := mapFromSlot(extension.HookSampleAttributes, slot); err == nil {
		t.Fatalf("expected error when non-core plugin payload present")
	}
}
