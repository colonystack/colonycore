package core

import (
	"testing"
	"time"

	"colonycore/pkg/domain"
)

// TestOrganismViewAttributesDeepCopy ensures that nested reference types returned
// from Attributes() cannot mutate the underlying stored attributes in the view.
func TestOrganismViewAttributesDeepCopy(t *testing.T) {

	original := map[string]any{
		"level1_map":   map[string]any{"k": "v", "nested_slice": []any{"a", map[string]any{"inner": "x"}}},
		"level1_slice": []any{map[string]any{"m": "n"}, []string{"s1", "s2"}},
	}

	org := domainOrganismFixture() // use existing test helper if available; fallback create minimal
	org.SetAttributes(original)

	view := newOrganismView(org)

	attrs := view.Attributes()
	// mutate top-level map
	attrs["new_key"] = "new_val"
	// mutate nested map
	lm := attrs["level1_map"].(map[string]any)
	lm["k"] = "changed"
	// mutate nested slice element's inner map
	ns := lm["nested_slice"].([]any)
	inner := ns[1].(map[string]any)
	inner["inner"] = "changed_inner"
	// mutate level1_slice nested structures
	ls := attrs["level1_slice"].([]any)
	lsMap := ls[0].(map[string]any)
	lsMap["m"] = "updated"
	strSlice := ls[1].([]string)
	if len(strSlice) > 0 {
		strSlice[0] = "mutated"
	}

	// Fetch again to ensure underlying data remains unchanged
	attrs2 := view.Attributes()

	// Assert original underlying data unaffected
	if _, exists := attrs2["new_key"]; exists {
		t.Fatalf("unexpected key 'new_key' present in second retrieval, clone not deep enough")
	}
	l1m2 := attrs2["level1_map"].(map[string]any)
	if l1m2["k"] != "v" {
		t.Errorf("expected nested map value 'v', got %v", l1m2["k"])
	}
	ns2 := l1m2["nested_slice"].([]any)
	inner2 := ns2[1].(map[string]any)
	if inner2["inner"] != "x" {
		t.Errorf("expected inner map value 'x', got %v", inner2["inner"])
	}
	l1s2 := attrs2["level1_slice"].([]any)
	l1s2Map := l1s2[0].(map[string]any)
	if l1s2Map["m"] != "n" {
		t.Errorf("expected level1_slice map value 'n', got %v", l1s2Map["m"])
	}
	strSlice2 := l1s2[1].([]string)
	if strSlice2[0] != "s1" {
		t.Errorf("expected string slice first element 's1', got %v", strSlice2[0])
	}
}

// domainOrganismFixture provides a minimal domain.Organism; replaced if a helper exists.
func domainOrganismFixture() domain.Organism {
	now := time.Now().UTC()
	return domain.Organism{Base: domain.Base{ID: "o1", CreatedAt: now, UpdatedAt: now}}
}

func TestCoreAttributesHelperClone(t *testing.T) {
	var org domain.Organism
	if attrs := org.CoreAttributes(); attrs != nil {
		t.Fatalf("expected nil core attributes for zero-value organism")
	}

	original := map[string]any{"flag": true}
	org.SetAttributes(original)

	values := org.CoreAttributes()
	if values["flag"] != true {
		t.Fatalf("expected cloned payload to include flag")
	}
	values["flag"] = false
	result := org.CoreAttributes()
	if result["flag"] != true {
		t.Fatalf("expected subsequent clone to remain unchanged")
	}
}
