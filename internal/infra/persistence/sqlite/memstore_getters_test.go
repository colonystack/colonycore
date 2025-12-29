package sqlite

import "testing"

func TestMemStoreGetMissing(t *testing.T) {
	store := newMemStore(nil)
	cases := []struct {
		name string
		fn   func() bool
	}{
		{"organism", func() bool { _, ok := store.GetOrganism("missing"); return ok }},
		{"housing", func() bool { _, ok := store.GetHousingUnit("missing"); return ok }},
		{"facility", func() bool { _, ok := store.GetFacility("missing"); return ok }},
		{"line", func() bool { _, ok := store.GetLine("missing"); return ok }},
		{"strain", func() bool { _, ok := store.GetStrain("missing"); return ok }},
		{"marker", func() bool { _, ok := store.GetGenotypeMarker("missing"); return ok }},
		{"permit", func() bool { _, ok := store.GetPermit("missing"); return ok }},
	}

	for _, tc := range cases {
		if tc.fn() {
			t.Fatalf("expected %s lookup to return false", tc.name)
		}
	}
}
