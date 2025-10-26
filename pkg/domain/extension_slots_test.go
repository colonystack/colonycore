package domain

import "testing"

func TestOrganismSetAttributesClonesInputAndOutput(t *testing.T) {
	original := map[string]any{
		"nested": []any{map[string]any{"k": "v"}},
	}
	var organism Organism
	organism.SetAttributes(original)

	// Mutate original input; stored value should remain unchanged.
	original["nested"].([]any)[0].(map[string]any)["k"] = "mutated"
	if organism.Attributes["nested"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored attributes to remain unchanged")
	}

	// Mutate returned map; stored value should remain unchanged.
	view := organism.AttributesMap()
	view["nested"].([]any)[0].(map[string]any)["k"] = "changed"
	if organism.Attributes["nested"].([]any)[0].(map[string]any)["k"] != "v" {
		t.Fatalf("expected stored attributes to remain unchanged after view mutation")
	}
}

func TestFacilityEnvironmentBaselinesHelpersClone(t *testing.T) {
	input := map[string]any{"temp": []int{20}}

	var facility Facility
	facility.SetEnvironmentBaselines(input)

	input["temp"].([]int)[0] = 99
	if facility.EnvironmentBaselines["temp"].([]int)[0] != 20 {
		t.Fatalf("expected facility baselines to be cloned from input")
	}

	baselines := facility.EnvironmentBaselinesMap()
	baselines["temp"].([]int)[0] = 30
	if facility.EnvironmentBaselines["temp"].([]int)[0] != 20 {
		t.Fatalf("expected facility baselines to remain unchanged after view mutation")
	}
}

func TestSampleSetAttributesNil(t *testing.T) {
	var sample Sample
	sample.SetAttributes(nil)
	if sample.Attributes != nil {
		t.Fatalf("expected nil attributes when input is nil")
	}
	if sample.AttributesMap() != nil {
		t.Fatalf("expected nil attribute map when stored map is nil")
	}
}
