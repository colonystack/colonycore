package datasetapi

import "testing"

func TestExtensionHookContextReferences(t *testing.T) {
	ctx := NewExtensionHookContext()
	hooks := []HookRef{
		ctx.OrganismAttributes(),
		ctx.FacilityEnvironmentBaselines(),
		ctx.BreedingUnitPairingAttributes(),
		ctx.ObservationData(),
		ctx.SampleAttributes(),
		ctx.SupplyItemAttributes(),
	}
	for i, hook := range hooks {
		if hook.String() == "" {
			t.Fatalf("hook %d returned empty identifier", i)
		}
		if !hook.Equals(hook) {
			t.Fatalf("hook %d should equal itself", i)
		}
		value := hook.(hookRef)
		if value.value() != hook.String() {
			t.Fatalf("hook %d string/value mismatch %q vs %q", i, value.value(), hook.String())
		}
		if !hook.Equals(&value) {
			t.Fatalf("hook %d should equal pointer variant", i)
		}
	}
	if hooks[0].Equals(hooks[1]) {
		t.Fatalf("distinct hooks should not be equal")
	}
}

func TestExtensionContributorContextReferences(t *testing.T) {
	ctx := NewExtensionContributorContext()
	core := ctx.Core()
	if core.String() != "core" {
		t.Fatalf("expected core identifier 'core', got %q", core.String())
	}
	if !core.Equals(core) {
		t.Fatalf("core reference should equal itself")
	}
	coreValue := core.(pluginRef)
	if coreValue.value() != "core" {
		t.Fatalf("expected core internal identifier, got %q", coreValue.value())
	}
	if !core.Equals(&coreValue) {
		t.Fatalf("core reference should equal pointer variant")
	}

	custom := ctx.Custom("custom.plugin")
	if custom.String() != "custom.plugin" {
		t.Fatalf("expected custom identifier, got %q", custom.String())
	}
	if custom.Equals(core) || core.Equals(custom) {
		t.Fatalf("custom plugin should not equal core")
	}
	if custom.(pluginRef).value() != "custom.plugin" {
		t.Fatalf("expected custom internal identifier, got %q", custom.(pluginRef).value())
	}
}
