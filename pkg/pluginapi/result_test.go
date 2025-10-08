package pluginapi

import "testing"

// TestResultConstructionAndAccessors exercises NewResult, NewViolation, accessors, and defensive copying.
func TestResultConstructionAndAccessors(t *testing.T) {
	// empty constructor path
	empty := NewResult()
	if empty.HasBlocking() {
		t.Fatalf("empty result should not have blocking violations")
	}
	if vols := empty.Violations(); vols != nil { // should be nil slice for empty
		t.Fatalf("expected nil violations slice, got %v", vols)
	}

	sev := NewSeverityContext().Warn()
	ent := NewEntityContext().Organism()
	v1 := NewViolation("rule-a", sev, "warn msg", ent, "O1")
	if v1.Rule() != "rule-a" || v1.Severity() != severityWarn || v1.Message() != "warn msg" || v1.Entity() != entityOrganism || v1.EntityID() != "O1" {
		t.Fatalf("violation accessor mismatch: %#v", v1)
	}

	r1 := NewResult(v1)
	if len(r1.Violations()) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(r1.Violations()))
	}

	// defensive copy: mutate returned slice reference should not affect internal state length
	vols := r1.Violations()
	if len(vols) != 1 { // baseline
		t.Fatalf("expected copy length 1")
	}
	_ = append(vols, v1) // append to local copy; original should remain length 1
	if len(r1.Violations()) != 1 {
		t.Fatalf("internal slice mutated via external append copy")
	}

	// AddViolation initial nil slice path
	added := Result{}.AddViolation(v1)
	if len(added.Violations()) != 1 {
		t.Fatalf("expected 1 violation after AddViolation, got %d", len(added.Violations()))
	}

	// AddViolation append path
	sev2 := NewSeverityContext().Block()
	ent2 := NewEntityContext().Protocol()
	v2 := NewViolation("rule-b", sev2, "block msg", ent2, "P1")
	appended := added.AddViolation(v2)
	if !appended.HasBlocking() {
		t.Fatalf("expected blocking violation detected")
	}
	if len(appended.Violations()) != 2 {
		t.Fatalf("expected 2 violations after append, got %d", len(appended.Violations()))
	}

	// Merge scenarios
	merged := Result{}.Merge(appended)
	if len(merged.Violations()) != 2 {
		t.Fatalf("merge from empty should return other, got %d", len(merged.Violations()))
	}
	// merge both non-empty
	double := appended.Merge(appended)
	if len(double.Violations()) != 4 {
		t.Fatalf("expected 4 violations after merging same result twice, got %d", len(double.Violations()))
	}
}
