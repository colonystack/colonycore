package datasetapi

import (
	"testing"
	"time"
)

func TestFacilityContextBehavior(t *testing.T) {
	ctx := NewFacilityContext()
	zones := ctx.Zones()
	access := ctx.AccessPolicies()

	if !zones.Biosecure().IsBiosecure() {
		t.Fatal("Biosecure zone should report IsBiosecure")
	}
	if !zones.Quarantine().IsQuarantine() {
		t.Fatal("Quarantine zone should report IsQuarantine")
	}
	if zones.General().IsBiosecure() {
		t.Fatal("General zone should not report biosecure")
	}
	if zones.Biosecure().Equals(zones.General()) {
		t.Fatal("Distinct zone references should not be equal")
	}
	if !zones.Biosecure().Equals(zones.Biosecure()) {
		t.Fatal("Identical zone references should compare equal")
	}
	if zones.Biosecure().String() == "" {
		t.Fatal("Zone string representation should not be empty")
	}

	if !access.Restricted().IsRestricted() {
		t.Fatal("Restricted policy should report restricted")
	}
	if !access.Open().AllowsVisitors() {
		t.Fatal("Open policy should allow visitors")
	}
	if access.StaffOnly().AllowsVisitors() {
		t.Fatal("Staff-only policy should not allow visitors")
	}
	if !access.Restricted().Equals(access.Restricted()) {
		t.Fatal("Restricted policy references should compare equal")
	}
	if access.Open().String() == "" {
		t.Fatal("Access policy string should not be empty")
	}
}

func TestTreatmentContextStatuses(t *testing.T) {
	statuses := NewTreatmentContext().Statuses()

	if !statuses.InProgress().IsActive() {
		t.Fatal("In-progress status should be active")
	}
	if !statuses.Completed().IsCompleted() {
		t.Fatal("Completed status should be completed")
	}
	if !statuses.Flagged().IsFlagged() {
		t.Fatal("Flagged status should report flagged")
	}
	if statuses.Planned().IsFlagged() {
		t.Fatal("Planned status should not report flagged")
	}
	if statuses.Flagged().String() == "" {
		t.Fatal("Treatment status string should not be empty")
	}
	if !statuses.Completed().Equals(statuses.Completed()) {
		t.Fatal("Equal treatment status references should compare equal")
	}
	if statuses.Planned().Equals(statuses.Flagged()) {
		t.Fatal("Different treatment statuses should not compare equal")
	}
}

func TestObservationContextShapes(t *testing.T) {
	shapes := NewObservationContext().Shapes()

	if !shapes.Structured().HasStructuredPayload() {
		t.Fatal("Structured shape should have structured payload")
	}
	if !shapes.Narrative().HasNarrativeNotes() {
		t.Fatal("Narrative shape should have narrative notes")
	}
	if !shapes.Mixed().HasStructuredPayload() || !shapes.Mixed().HasNarrativeNotes() {
		t.Fatal("Mixed shape should report both payload types")
	}
	if shapes.Narrative().String() == "" {
		t.Fatal("Observation shape string should not be empty")
	}
	if shapes.Structured().Equals(shapes.Mixed()) {
		t.Fatal("Different observation shapes should not compare equal")
	}
}

func TestSampleContextSemantics(t *testing.T) {
	ctx := NewSampleContext()
	sources := ctx.Sources()
	statuses := ctx.Statuses()

	if !sources.Organism().IsOrganismDerived() {
		t.Fatal("Organism source should report organism derived")
	}
	if !sources.Cohort().IsCohortDerived() {
		t.Fatal("Cohort source should report cohort derived")
	}
	if !sources.Environmental().IsEnvironmental() {
		t.Fatal("Environmental source should report environmental")
	}
	if sources.Unknown().String() != "unknown" {
		t.Fatal("Unknown source should return literal string")
	}
	if !sources.Cohort().Equals(sources.Cohort()) {
		t.Fatal("Identical sample sources should compare equal")
	}

	if !statuses.Stored().IsAvailable() {
		t.Fatal("Stored status should be available")
	}
	if !statuses.InTransit().IsAvailable() {
		t.Fatal("In-transit status should be available")
	}
	if !statuses.Consumed().IsTerminal() {
		t.Fatal("Consumed status should be terminal")
	}
	if !statuses.Disposed().IsTerminal() {
		t.Fatal("Disposed status should be terminal")
	}
	if statuses.Stored().String() == "" {
		t.Fatal("Sample status string should not be empty")
	}
	if !statuses.Stored().Equals(statuses.Stored()) {
		t.Fatal("Stored statuses should compare equal")
	}
	if statuses.Stored().Equals(statuses.Disposed()) {
		t.Fatal("Different sample statuses should not compare equal")
	}
}

func TestPermitContextStatuses(t *testing.T) {
	statuses := NewPermitContext().Statuses()

	if !statuses.Submitted().Equals(statuses.Submitted()) {
		t.Fatal("Submitted references should be equal")
	}
	if statuses.Draft().String() == "" {
		t.Fatal("Permit status string should not be empty")
	}
	if statuses.Draft().Equals(statuses.Approved()) {
		t.Fatal("Different permit statuses should not compare equal")
	}
	if !statuses.Approved().IsActive() {
		t.Fatal("Approved status should report active")
	}
	if !statuses.Expired().IsExpired() {
		t.Fatal("Expired status should report expired")
	}
	if !statuses.Archived().IsArchived() {
		t.Fatal("Archived status should report archived")
	}
}

func TestSupplyContextStatuses(t *testing.T) {
	statuses := NewSupplyContext().Statuses()
	now := time.Now()
	past := now.Add(-time.Hour)

	if statuses.Healthy().IsExpired() {
		t.Fatal("Healthy status should not report expired")
	}
	if !statuses.Reorder().RequiresReorder() {
		t.Fatal("Reorder status should require reorder")
	}
	if !statuses.Critical().RequiresReorder() {
		t.Fatal("Critical status should require reorder")
	}
	if !statuses.Expired().IsExpired() {
		t.Fatal("Expired status should report expired")
	}
	if !statuses.Critical().Equals(statuses.Critical()) {
		t.Fatal("Critical references should be equal")
	}
	if statuses.Critical().Equals(statuses.Healthy()) {
		t.Fatal("Different status references should not be equal")
	}
	if statuses.Healthy().String() == "" {
		t.Fatal("Supply status string should not be empty")
	}
	if statuses.Expired().Equals(statuses.Critical()) {
		t.Fatal("Different supply statuses should not compare equal")
	}

	// ensure the helper signature remains used so that lint tools observe it
	_ = past
}
