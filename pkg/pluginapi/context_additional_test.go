package pluginapi

import (
	"testing"
	"time"
)

func TestFacilityContext(t *testing.T) {
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
		t.Fatal("Distinct zone references should not compare equal")
	}
	if zones.Biosecure().String() == "" {
		t.Fatal("Zone string representation should not be empty")
	}

	if !access.Restricted().IsRestricted() {
		t.Fatal("Restricted policy should report IsRestricted")
	}
	if !access.Open().AllowsVisitors() {
		t.Fatal("Open policy should allow visitors")
	}
	if access.StaffOnly().AllowsVisitors() {
		t.Fatal("Staff-only policy should not allow visitors")
	}
	if !access.Restricted().Equals(access.Restricted()) {
		t.Fatal("Restricted access policies should compare equal")
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
	if statuses.Planned().IsCompleted() {
		t.Fatal("Planned status should not be completed")
	}
	if statuses.Flagged().String() == "" {
		t.Fatal("Treatment status string should not be empty")
	}
	if !statuses.Completed().Equals(statuses.Completed()) {
		t.Fatal("Equal treatment statuses should compare equal")
	}
}

func TestObservationContextShapes(t *testing.T) {
	shapes := NewObservationContext().Shapes()

	if !shapes.Structured().HasStructuredPayload() {
		t.Fatal("Structured shape should report structured payload")
	}
	if !shapes.Narrative().HasNarrativeNotes() {
		t.Fatal("Narrative shape should report narrative notes")
	}
	if !shapes.Mixed().HasStructuredPayload() || !shapes.Mixed().HasNarrativeNotes() {
		t.Fatal("Mixed shape should report both structured payload and narrative notes")
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
}

func TestPermitContextStatuses(t *testing.T) {
	statuses := NewPermitContext().Statuses()

	if !statuses.Active().IsActive() {
		t.Fatal("Active status should report active")
	}
	if !statuses.Expired().IsExpired() {
		t.Fatal("Expired status should report expired")
	}
	if statuses.Pending().IsExpired() {
		t.Fatal("Pending status should not be expired")
	}
	if statuses.Pending().String() == "" {
		t.Fatal("Permit status string should not be empty")
	}
	if !statuses.Pending().Equals(statuses.Pending()) {
		t.Fatal("Identical permit statuses should compare equal")
	}
}

func TestSupplyContextStatuses(t *testing.T) {
	statuses := NewSupplyContext().Statuses()
	now := time.Now()
	expires := now.Add(-time.Hour)

	if statuses.Healthy().RequiresReorder() {
		t.Fatal("Healthy status should not require reorder")
	}
	if !statuses.Reorder().RequiresReorder() {
		t.Fatal("Reorder status should require reorder")
	}
	if statuses.Critical().IsExpired() {
		t.Fatal("Critical status should not imply expired")
	}
	if !statuses.Expired().IsExpired() {
		t.Fatal("Expired status should report expired")
	}
	if !statuses.Reorder().Equals(statuses.Reorder()) {
		t.Fatal("Reorder references should be equal")
	}
	if statuses.Reorder().Equals(statuses.Healthy()) {
		t.Fatal("Different status references should not be equal")
	}
	if statuses.Healthy().String() == "" {
		t.Fatal("Supply status string should not be empty")
	}

	_ = expires
}

func TestContextEqualityHelpers(t *testing.T) {
	entities := NewEntityContext()
	if entities.Organism().Equals(entities.Housing()) {
		t.Fatal("distinct entity refs should not compare equal")
	}
	if !entities.Organism().Equals(entities.Organism()) {
		t.Fatal("identical entity refs should compare equal")
	}
	if !entities.Organism().IsCore() || !entities.Protocol().IsCore() {
		t.Fatal("core entities should report IsCore")
	}

	actions := NewActionContext()
	if actions.Create().Equals(actions.Update()) {
		t.Fatal("different action refs should not compare equal")
	}
	if !actions.Create().Equals(actions.Create()) {
		t.Fatal("identical action refs should compare equal")
	}

	severities := NewSeverityContext()
	if severities.Warn().Equals(severities.Block()) {
		t.Fatal("different severity refs should not compare equal")
	}
	if !severities.Warn().Equals(severities.Warn()) {
		t.Fatal("identical severity refs should compare equal")
	}
}
