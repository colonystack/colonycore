package datasetapi

import (
	"testing"
	"time"
)

// TestDeepCloneMapSlice exercises cloning of []map[string]any branch indirectly via Organism.Attributes.
func TestDeepCloneMapSlice(t *testing.T) {
	attrs := map[string]any{
		"maps": []map[string]any{{"k": "v"}, {}},
	}
	org := NewOrganism(OrganismData{Base: BaseData{ID: "c", CreatedAt: time.Now().UTC()}, Attributes: attrs})
	ret := org.Attributes()
	// mutate returned
	ret["maps"].([]map[string]any)[0]["k"] = "mut"
	// ensure original still pristine via fresh call
	fresh := org.Attributes()["maps"].([]map[string]any)[0]["k"].(string)
	if fresh != "v" {
		t.Fatalf("expected original value 'v', got %s", fresh)
	}
}
