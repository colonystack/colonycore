package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

const testMutated = "mutated"

func TestLineEnsureSlots(t *testing.T) {
	line := &Line{}
	defaultSlot := line.EnsureDefaultAttributes()
	if err := defaultSlot.Set(extension.PluginCore, map[string]any{"seed": "value"}); err != nil {
		t.Fatalf("set default slot: %v", err)
	}
	if err := line.SetDefaultAttributesSlot(defaultSlot); err != nil {
		t.Fatalf("SetDefaultAttributesSlot: %v", err)
	}
	ensureDefault := line.EnsureDefaultAttributes()
	if ensureDefault.Hook() != extension.HookLineDefaultAttributes {
		t.Fatalf("expected default hook %q, got %q", extension.HookLineDefaultAttributes, ensureDefault.Hook())
	}

	overrides := line.EnsureExtensionOverrides()
	if err := overrides.Set(extension.PluginCore, map[string]any{"toggle": true}); err != nil {
		t.Fatalf("set overrides: %v", err)
	}
	if err := line.SetExtensionOverridesSlot(overrides); err != nil {
		t.Fatalf("SetExtensionOverridesSlot: %v", err)
	}
	ensureOverrides := line.EnsureExtensionOverrides()
	if ensureOverrides.Hook() != extension.HookLineExtensionOverrides {
		t.Fatalf("expected overrides hook %q, got %q", extension.HookLineExtensionOverrides, ensureOverrides.Hook())
	}
}

func TestLineSetDefaultAttributesSlotClearsOnNil(t *testing.T) {
	var line Line
	slot := extension.NewSlot(extension.HookLineDefaultAttributes)
	if err := slot.Set(extension.PluginCore, map[string]any{"seed": true}); err != nil {
		t.Fatalf("set default slot: %v", err)
	}
	if err := line.SetDefaultAttributesSlot(slot); err != nil {
		t.Fatalf("SetDefaultAttributesSlot: %v", err)
	}
	if line.extensions == nil {
		t.Fatalf("expected extensions initialised")
	}
	if err := line.SetDefaultAttributesSlot(nil); err != nil {
		t.Fatalf("SetDefaultAttributesSlot nil: %v", err)
	}
	if line.extensions != nil {
		t.Fatalf("expected extensions cleared after nil")
	}
}
