package memory

import "testing"

func TestContainsStringCoverage(t *testing.T) {
	if !containsString([]string{"alpha", "beta"}, "beta") {
		t.Fatalf("expected containsString to return true when element exists")
	}
	if containsString([]string{"alpha", "beta"}, "gamma") {
		t.Fatalf("expected containsString to return false when element missing")
	}
}

func TestTransactionViewFindersMissingEntities(t *testing.T) {
	state := newMemoryState()
	view := transactionView{state: &state}

	checks := []struct {
		name string
		ok   bool
	}{
		{"FindOrganism", func() bool { _, ok := view.FindOrganism("missing"); return ok }()},
		{"FindHousingUnit", func() bool { _, ok := view.FindHousingUnit("missing"); return ok }()},
		{"FindFacility", func() bool { _, ok := view.FindFacility("missing"); return ok }()},
		{"FindLine", func() bool { _, ok := view.FindLine("missing"); return ok }()},
		{"FindStrain", func() bool { _, ok := view.FindStrain("missing"); return ok }()},
		{"FindGenotypeMarker", func() bool { _, ok := view.FindGenotypeMarker("missing"); return ok }()},
	}

	for _, check := range checks {
		if check.ok {
			t.Fatalf("expected %s to return ok=false for unknown ID", check.name)
		}
	}
}

func TestTransactionFindersMissingEntities(t *testing.T) {
	tx := &transaction{state: newMemoryState()}

	checks := []struct {
		name string
		ok   bool
	}{
		{"FindHousingUnit", func() bool { _, ok := tx.FindHousingUnit("missing"); return ok }()},
		{"FindProtocol", func() bool { _, ok := tx.FindProtocol("missing"); return ok }()},
		{"FindFacility", func() bool { _, ok := tx.FindFacility("missing"); return ok }()},
		{"FindLine", func() bool { _, ok := tx.FindLine("missing"); return ok }()},
		{"FindStrain", func() bool { _, ok := tx.FindStrain("missing"); return ok }()},
		{"FindGenotypeMarker", func() bool { _, ok := tx.FindGenotypeMarker("missing"); return ok }()},
		{"FindTreatment", func() bool { _, ok := tx.FindTreatment("missing"); return ok }()},
		{"FindObservation", func() bool { _, ok := tx.FindObservation("missing"); return ok }()},
		{"FindSample", func() bool { _, ok := tx.FindSample("missing"); return ok }()},
		{"FindPermit", func() bool { _, ok := tx.FindPermit("missing"); return ok }()},
		{"FindSupplyItem", func() bool { _, ok := tx.FindSupplyItem("missing"); return ok }()},
	}

	for _, check := range checks {
		if check.ok {
			t.Fatalf("expected %s to return ok=false for unknown ID", check.name)
		}
	}
}
