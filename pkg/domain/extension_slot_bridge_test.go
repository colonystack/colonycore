package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

func TestSlotFromContainerRoundTrip(t *testing.T) {
	var organism Organism
	mustNoError(t, "SetCoreAttributes", organism.SetCoreAttributes(map[string]any{"flag": true}))

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
