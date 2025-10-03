package pluginapi

import "testing"

func TestChangeSnapshotDeepClone(t *testing.T) {
	before := map[string]any{
		"outer": []any{map[string]any{"k": "v"}, []string{"x", "y"}},
	}
	ch := NewChange(EntityOrganism, ActionUpdate, before, nil)
	// mutate after construction deeply
	before["outer"].([]any)[0].(map[string]any)["k"] = "mutated"
	before["outer"].([]any)[1].([]string)[0] = "z"
	// mutate accessor result deeply
	acc := ch.Before().(map[string]any)
	accOuter := acc["outer"].([]any)
	accOuter[0].(map[string]any)["k"] = "accessor-mutated"
	accOuter[1].([]string)[0] = "w"
	// fetch again to verify immutability
	snap := ch.Before().(map[string]any)
	outer := snap["outer"].([]any)
	innerMap := outer[0].(map[string]any)
	if innerMap["k"].(string) != "v" {
		t.Fatalf("expected deep-cloned inner map value 'v', got %v", innerMap["k"])
	}
	innerSlice := outer[1].([]string)
	if innerSlice[0] != "x" {
		t.Fatalf("expected deep-cloned inner slice element 'x', got %s", innerSlice[0])
	}
}

func TestSnapshotValuePrimitiveScalars(t *testing.T) {
	primitives := []any{uint64(5), float32(3.14), true, "text"}
	ch := NewChange(EntityProject, ActionUpdate, primitives, nil)
	got := ch.Before().([]any)
	if len(got) != 4 {
		t.Fatalf("expected 4 primitives, got %d", len(got))
	}
	if _, ok := got[0].(uint64); !ok {
		t.Fatalf("expected uint64 preserved, got %T", got[0])
	}
	if _, ok := got[1].(float64); !ok { // float32 marshals to float64 via interface widening
		// Note: float32 is stored as float32 originally but when placed in []any it keeps type; snapshot returns directly.
		// Accept either float32 or float64.
		if _, alt := got[1].(float32); !alt {
			t.Fatalf("expected float32/float64, got %T", got[1])
		}
	}
	if v, ok := got[2].(bool); !ok || !v {
		t.Fatalf("expected bool true, got %v (%T)", got[2], got[2])
	}
	if v, ok := got[3].(string); !ok || v != "text" {
		t.Fatalf("expected string 'text', got %v (%T)", got[3], got[3])
	}
}
