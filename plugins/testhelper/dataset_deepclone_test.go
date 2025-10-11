package testhelper

import (
	"testing"
	"time"
)

// TestDeepCloneAttrEmptyBranches ensures empty map/slice branches produce distinct non-nil instances.
func TestDeepCloneAttrEmptyBranches(t *testing.T) {
	now := time.Now().UTC()
	emptyAttrs := map[string]any{
		"emptymap":   map[string]any{},
		"emptyslice": []any{},
		"emptystrs":  []string{},
		"emptymaps":  []map[string]any{},
	}
	org := Organism(OrganismFixtureConfig{
		BaseFixture: BaseFixture{ID: "empty", CreatedAt: now, UpdatedAt: now},
		Attributes:  emptyAttrs,
		Stage:       LifecycleStages().Adult,
	})
	attrs := org.Attributes()
	for k, v := range attrs {
		switch k {
		case "emptymap":
			if m, ok := v.(map[string]any); !ok || m == nil || len(m) != 0 {
				t.Fatalf("expected empty map for %s", k)
			}
		case "emptyslice":
			if s, ok := v.([]any); !ok || s == nil || len(s) != 0 {
				t.Fatalf("expected empty []any for %s", k)
			}
		case "emptystrs":
			if s, ok := v.([]string); !ok || s == nil || len(s) != 0 {
				t.Fatalf("expected empty []string for %s", k)
			}
		case "emptymaps":
			if s, ok := v.([]map[string]any); !ok || s == nil || len(s) != 0 {
				t.Fatalf("expected empty []map for %s", k)
			}
		}
	}
}
