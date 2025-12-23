package pluginapi

import "testing"

func TestChangeSnapshotDeepClone(t *testing.T) {
	before := map[string]any{
		"outer": []any{map[string]any{"k": "v"}, []string{"x", "y"}},
	}
	ch := newChangeForTest(entityOrganism, actionUpdate, newPayload(t, before), UndefinedChangePayload())
	// mutate after construction deeply
	before["outer"].([]any)[0].(map[string]any)["k"] = "mutated"
	before["outer"].([]any)[1].([]string)[0] = "z"
	// mutate accessor result deeply
	var acc map[string]any
	unmarshalPayload(t, ch.Before(), &acc)
	accOuter := acc["outer"].([]any)
	accOuter[0].(map[string]any)["k"] = "accessor-mutated"
	accOuterSlice := accOuter[1].([]any)
	accOuterSlice[0] = "w"
	// fetch again to verify immutability
	var snap map[string]any
	unmarshalPayload(t, ch.Before(), &snap)
	outer := snap["outer"].([]any)
	innerMap := outer[0].(map[string]any)
	if innerMap["k"].(string) != "v" {
		t.Fatalf("expected deep-cloned inner map value 'v', got %v", innerMap["k"])
	}
	innerSlice := outer[1].([]any)
	if innerSlice[0].(string) != "x" {
		t.Fatalf("expected deep-cloned inner slice element 'x', got %v", innerSlice[0])
	}
}

func TestSnapshotValuePrimitiveScalars(t *testing.T) {
	primitives := []any{uint64(5), float32(3.14), true, "text"}
	ch := newChangeForTest(entityProject, actionUpdate, newPayload(t, primitives), UndefinedChangePayload())
	var got []any
	unmarshalPayload(t, ch.Before(), &got)
	if len(got) != 4 {
		t.Fatalf("expected 4 primitives, got %d", len(got))
	}
	if v, ok := got[0].(float64); !ok || v != 5 {
		t.Fatalf("expected float64 5, got %v (%T)", got[0], got[0])
	}
	if v, ok := got[1].(float64); !ok || v != 3.14 {
		t.Fatalf("expected float64 3.14, got %v (%T)", got[1], got[1])
	}
	if v, ok := got[2].(bool); !ok || !v {
		t.Fatalf("expected bool true, got %v (%T)", got[2], got[2])
	}
	if v, ok := got[3].(string); !ok || v != "text" {
		t.Fatalf("expected string 'text', got %v (%T)", got[3], got[3])
	}
}
