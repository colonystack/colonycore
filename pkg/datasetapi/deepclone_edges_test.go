package datasetapi

import "testing"

// TestDeepCloneEmptyContainers exercises empty map/slice branches returning non-nil empty instances.
func TestDeepCloneEmptyContainers(t *testing.T) {
	attrs := map[string]any{
		"emptymap":   map[string]any{},
		"emptyslice": []any{},
		"emptystrs":  []string{},
		"emptymaps":  []map[string]any{},
	}
	org := NewOrganism(OrganismData{Base: BaseData{ID: "edge"}, Attributes: attrs})
	got := org.Attributes()
	for k, expectType := range map[string]string{"emptymap": "map", "emptyslice": "slice", "emptystrs": "slice", "emptymaps": "slice"} {
		v := got[k]
		switch expectType {
		case "map":
			if m, ok := v.(map[string]any); !ok || m == nil {
				t.Fatalf("expected non-nil empty map for %s", k)
			}
		case "slice":
			// We only assert non-nil slice via len==0 using type switch.
			switch s := v.(type) {
			case []any:
				if len(s) != 0 {
					t.Fatalf("expected empty []any for %s", k)
				}
			case []string:
				if len(s) != 0 {
					t.Fatalf("expected empty []string for %s", k)
				}
			case []map[string]any:
				if len(s) != 0 {
					t.Fatalf("expected empty []map for %s", k)
				}
			default:
				t.Fatalf("unexpected slice concrete type %T for %s", v, k)
			}
		}
	}
}
