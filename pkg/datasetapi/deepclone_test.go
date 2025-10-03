package datasetapi

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDeepCloneAdditionalBranches exercises deepClone branches for []map[string]any, []any and []string.
func TestDeepCloneAdditionalBranches(t *testing.T) {
	nested := map[string]any{
		"mapslice":   []map[string]any{{"k": "v"}, {}},
		"mixed":      []any{[]string{"a", "b"}, map[string]any{"inner": []string{"x"}}},
		"emptyMap":   map[string]any{},
		"emptySlice": []any{},
	}
	org := NewOrganism(OrganismData{Base: BaseData{ID: "clone", CreatedAt: time.Now().UTC()}, Attributes: nested})
	attrs := org.Attributes()
	attrs["mapslice"].([]map[string]any)[0]["k"] = "mutated"
	attrs["mixed"].([]any)[0].([]string)[0] = "z"
	fresh := org.Attributes()
	if fresh["mapslice"].([]map[string]any)[0]["k"].(string) != "v" {
		t.Fatalf("expected nested map value to remain 'v'")
	}
	if fresh["mixed"].([]any)[0].([]string)[0] != "a" {
		t.Fatalf("expected nested []string element to remain 'a'")
	}
	if b, err := json.Marshal(org); err != nil {
		t.Fatalf("marshal organism: %v", err)
	} else if len(b) == 0 {
		t.Fatalf("expected non-empty json")
	}
}
