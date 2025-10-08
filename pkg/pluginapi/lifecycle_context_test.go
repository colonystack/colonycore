package pluginapi

import "testing"

func TestPluginAPILifecycleStageContext(t *testing.T) {
	ctx := NewLifecycleStageContext()

	// Test all stage accessors
	planned := ctx.Planned()
	larva := ctx.Larva()
	juvenile := ctx.Juvenile()
	adult := ctx.Adult()
	retired := ctx.Retired()
	deceased := ctx.Deceased()

	// Test string representations
	if planned.String() != "planned" {
		t.Errorf("expected planned stage string 'planned', got '%s'", planned.String())
	}
	if larva.String() != "embryo_larva" {
		t.Errorf("expected larva stage string 'embryo_larva', got '%s'", larva.String())
	}
	if juvenile.String() != "juvenile" {
		t.Errorf("expected juvenile stage string 'juvenile', got '%s'", juvenile.String())
	}
	if adult.String() != "adult" {
		t.Errorf("expected adult stage string 'adult', got '%s'", adult.String())
	}
	if retired.String() != "retired" {
		t.Errorf("expected retired stage string 'retired', got '%s'", retired.String())
	}
	if deceased.String() != "deceased" {
		t.Errorf("expected deceased stage string 'deceased', got '%s'", deceased.String())
	}

	// Test IsActive
	if !planned.IsActive() {
		t.Error("expected planned to be active")
	}
	if !larva.IsActive() {
		t.Error("expected larva to be active")
	}
	if !juvenile.IsActive() {
		t.Error("expected juvenile to be active")
	}
	if !adult.IsActive() {
		t.Error("expected adult to be active")
	}
	if retired.IsActive() {
		t.Error("expected retired to not be active")
	}
	if deceased.IsActive() {
		t.Error("expected deceased to not be active")
	}

	// Test Equals
	if !adult.Equals(ctx.Adult()) {
		t.Error("expected adult stages to be equal")
	}
	if adult.Equals(retired) {
		t.Error("expected adult and retired stages to not be equal")
	}

	// Test Value method (internal use)
	if adult.Value() != stageAdult {
		t.Errorf("expected adult value to be '%s', got '%s'", stageAdult, adult.Value())
	}
}

func TestPluginAPILifecycleStageRefEdgeCases(t *testing.T) {
	ctx := NewLifecycleStageContext()
	adult := ctx.Adult()

	// Test Equals with different type (should return false)
	if adult.Equals(nil) {
		t.Error("expected adult.Equals(nil) to return false")
	}

	// Test the internal marker method
	adult.isLifecycleStageRef() // Should not panic
}
