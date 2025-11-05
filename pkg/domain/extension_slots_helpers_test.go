package domain

import (
	"errors"
	"testing"

	"colonycore/pkg/domain/extension"
)

func TestSlotFromPluginPayloads(t *testing.T) {
	slot, err := slotFromPluginPayloads(extension.HookLineDefaultAttributes, nil)
	if err != nil {
		t.Fatalf("expected nil payload to succeed: %v", err)
	}
	if slot != nil {
		t.Fatalf("expected nil payload to return nil slot")
	}

	slot, err = slotFromPluginPayloads(extension.HookLineDefaultAttributes, map[string]any{})
	if err != nil {
		t.Fatalf("expected empty payload to succeed: %v", err)
	}
	if slot != nil {
		t.Fatalf("expected empty payload to return nil slot")
	}

	payload := map[string]any{
		extension.PluginCore.String(): map[string]any{"seed": "value"},
		"plugin.alpha":                map[string]any{"flag": true},
	}
	slot, err = slotFromPluginPayloads(extension.HookLineDefaultAttributes, payload)
	if err != nil {
		t.Fatalf("slotFromPluginPayloads: %v", err)
	}
	if slot == nil {
		t.Fatalf("expected slot to be created for populated payload")
	}
	core, ok := slot.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected core payload to be present")
	}
	if core.(map[string]any)["seed"] != "value" {
		t.Fatalf("unexpected core payload: %+v", core)
	}
	payload[extension.PluginCore.String()].(map[string]any)["seed"] = "mutated"
	recheck, ok := slot.Get(extension.PluginCore)
	if !ok {
		t.Fatalf("expected core payload after mutation")
	}
	if recheck.(map[string]any)["seed"] != "value" {
		t.Fatalf("expected slot payload to remain cloned, got %v", recheck)
	}
}

func TestSlotFromPluginPayloadsInvalid(t *testing.T) {
	_, err := slotFromPluginPayloads(extension.HookLineDefaultAttributes, map[string]any{
		"": map[string]any{},
	})
	if !errors.Is(err, extension.ErrEmptyPlugin) {
		t.Fatalf("expected ErrEmptyPlugin, got %v", err)
	}

	_, err = slotFromPluginPayloads(extension.HookLineDefaultAttributes, map[string]any{
		extension.PluginCore.String(): []string{"invalid"},
	})
	if err == nil {
		t.Fatalf("expected shape validation error")
	}
}
