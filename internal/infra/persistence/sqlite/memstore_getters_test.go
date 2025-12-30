package sqlite

import (
	"strings"
	"testing"
)

func TestMemStoreGetMissing(t *testing.T) {
	store := newMemStore(nil)
	longID := strings.Repeat("a", 1024)
	ids := []string{
		"missing",
		"",
		longID,
		"id/with/slash",
		"id:with:colon",
		"id?with?query",
		"id#with#hash",
	}
	cases := []struct {
		name string
		fn   func(string) bool
	}{
		{"organism", func(id string) bool { _, ok := store.GetOrganism(id); return ok }},
		{"housing", func(id string) bool { _, ok := store.GetHousingUnit(id); return ok }},
		{"facility", func(id string) bool { _, ok := store.GetFacility(id); return ok }},
		{"line", func(id string) bool { _, ok := store.GetLine(id); return ok }},
		{"strain", func(id string) bool { _, ok := store.GetStrain(id); return ok }},
		{"marker", func(id string) bool { _, ok := store.GetGenotypeMarker(id); return ok }},
		{"permit", func(id string) bool { _, ok := store.GetPermit(id); return ok }},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, id := range ids {
				if tc.fn(id) {
					t.Fatalf("expected %s lookup for %q to return false", tc.name, id)
				}
			}
		})
	}
}
