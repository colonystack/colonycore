package domain

import (
	"testing"

	"colonycore/pkg/domain/extension"
)

const testMutated = "mutated"

func TestLineAndStrainEnsureSlots(t *testing.T) {
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

	strain := &Strain{}
	slot := strain.EnsureAttributes()
	if err := slot.Set(extension.PluginCore, map[string]any{"note": "strain"}); err != nil {
		t.Fatalf("set strain slot: %v", err)
	}
	if err := strain.SetAttributesSlot(slot); err != nil {
		t.Fatalf("SetAttributesSlot (strain): %v", err)
	}
	if strain.attributesSlot == nil {
		t.Fatalf("expected strain slot rebound")
	}

	genotype := &GenotypeMarker{}
	gslot := genotype.EnsureAttributes()
	if err := gslot.Set(extension.PluginCore, map[string]any{"note": "geno"}); err != nil {
		t.Fatalf("set genotype slot: %v", err)
	}
	if err := genotype.SetAttributesSlot(gslot); err != nil {
		t.Fatalf("SetAttributesSlot (genotype): %v", err)
	}
	if genotype.attributesSlot == nil {
		t.Fatalf("expected genotype slot rebound")
	}

	// Simulate JSON unmarshalling that loses hook binding.
	unbound := extension.NewSlot("")
	strain.attributesSlot = unbound
	genotype.attributesSlot = unbound.Clone()
	if strain.EnsureAttributes().Hook() != extension.HookStrainAttributes {
		t.Fatalf("expected strain hook rebind")
	}
	if genotype.EnsureAttributes().Hook() != extension.HookGenotypeMarkerAttributes {
		t.Fatalf("expected genotype hook rebind")
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
