package pluginapi

import "testing"

func TestExtensionHookContextReferences(t *testing.T) {
	ctx := NewExtensionHookContext()
	all := []HookRef{
		ctx.OrganismAttributes(),
		ctx.FacilityEnvironmentBaselines(),
		ctx.BreedingUnitPairingAttributes(),
		ctx.ObservationData(),
		ctx.SampleAttributes(),
		ctx.SupplyItemAttributes(),
	}
	for i, ref := range all {
		if ref.String() == "" {
			t.Fatalf("hook %d returned empty identifier", i)
		}
		if !ref.Equals(ref) {
			t.Fatalf("hook %d should equal itself", i)
		}
		if ref.Equals(ctx.OrganismAttributes()) && i != 0 {
			t.Fatalf("hook %d unexpectedly matched organism hook", i)
		}
		value := ref.(hookRef)
		if !ref.Equals(&value) {
			t.Fatalf("hook %d should equal pointer to same value", i)
		}
		if value.value() == "" {
			t.Fatalf("hook %d value should not be empty", i)
		}
	}
}

func TestExtensionContributorContextReferences(t *testing.T) {
	ctx := NewExtensionContributorContext()
	core := ctx.Core()
	if core.String() != core.(pluginRef).value() {
		t.Fatalf("core plugin string mismatch: %q vs value()", core.String())
	}
	if !core.Equals(core) {
		t.Fatalf("core reference should equal itself")
	}
	coreValue := core.(pluginRef)
	if !core.Equals(&coreValue) {
		t.Fatalf("core reference should equal pointer variant")
	}

	custom := ctx.Custom("external.plugin")
	if custom.String() != "external.plugin" {
		t.Fatalf("expected custom plugin identifier, got %q", custom.String())
	}
	if custom.Equals(core) || core.Equals(custom) {
		t.Fatalf("custom plugin should not equal core plugin")
	}
	if custom.(pluginRef).value() != "external.plugin" {
		t.Fatalf("expected custom plugin internal value, got %q", custom.(pluginRef).value())
	}
}
