package pluginapi

import (
	"testing"
)

// TestActionRefChecker tests the isActionRef function
func TestActionRefChecker(_ *testing.T) {
	ctx := NewActionContext()
	create := ctx.Create()

	// Call the function to ensure it's covered
	create.isActionRef()
}

// TestEntityTypeRefChecker tests the isEntityTypeRef function
func TestEntityTypeRefChecker(_ *testing.T) {
	ctx := NewEntityContext()
	organism := ctx.Organism()

	// Call the function to ensure it's covered
	organism.isEntityTypeRef()
}

// TestFacilityZoneRefChecker tests the isFacilityZoneRef function
func TestFacilityZoneRefChecker(_ *testing.T) {
	ctx := NewFacilityContext()
	biosecure := ctx.Zones().Biosecure()

	// Call the function to ensure it's covered
	biosecure.isFacilityZoneRef()
}

// TestFacilityAccessPolicyRefChecker tests the isFacilityAccessPolicyRef function
func TestFacilityAccessPolicyRefChecker(_ *testing.T) {
	ctx := NewFacilityContext()
	restricted := ctx.AccessPolicies().Restricted()

	// Call the function to ensure it's covered
	restricted.isFacilityAccessPolicyRef()
}

// TestEnvironmentTypeRefChecker tests the isEnvironmentTypeRef function
func TestEnvironmentTypeRefChecker(_ *testing.T) {
	ctx := NewHousingContext()
	aquatic := ctx.Aquatic()

	// Call the function to ensure it's covered
	aquatic.isEnvironmentTypeRef()
}

// TestLifecycleStageRefChecker tests the isLifecycleStageRef function
func TestLifecycleStageRefChecker(_ *testing.T) {
	ctx := NewLifecycleStageContext()
	adult := ctx.Adult()

	// Call the function to ensure it's covered
	adult.isLifecycleStageRef()
}

// TestObservationShapeRefChecker tests the isObservationShapeRef function
func TestObservationShapeRefChecker(_ *testing.T) {
	ctx := NewObservationContext()
	narrative := ctx.Shapes().Narrative()

	// Call the function to ensure it's covered
	narrative.isObservationShapeRef()
}

// TestPermitStatusRefChecker tests the isPermitStatusRef function
func TestPermitStatusRefChecker(_ *testing.T) {
	ctx := NewPermitContext()
	approved := ctx.Statuses().Approved()

	// Call the function to ensure it's covered
	approved.isPermitStatusRef()
}

// TestProtocolStatusRefChecker tests the isProtocolStatusRef function
func TestProtocolStatusRefChecker(_ *testing.T) {
	ctx := NewProtocolContext()
	approved := ctx.Approved()

	// Call the function to ensure it's covered
	approved.isProtocolStatusRef()
}

// TestSampleSourceRefChecker tests the isSampleSourceRef function
func TestSampleSourceRefChecker(_ *testing.T) {
	ctx := NewSampleContext()
	organism := ctx.Sources().Organism()

	// Call the function to ensure it's covered
	organism.isSampleSourceRef()
}

// TestSampleStatusRefChecker tests the isSampleStatusRef function
func TestSampleStatusRefChecker(_ *testing.T) {
	ctx := NewSampleContext()
	stored := ctx.Statuses().Stored()

	// Call the function to ensure it's covered
	stored.isSampleStatusRef()
}

// TestSeverityRefChecker tests the isSeverityRef function
func TestSeverityRefChecker(_ *testing.T) {
	ctx := NewSeverityContext()
	warn := ctx.Warn()

	// Call the function to ensure it's covered
	warn.isSeverityRef()
}

// TestSupplyStatusRefChecker tests the isSupplyStatusRef function
func TestSupplyStatusRefChecker(_ *testing.T) {
	ctx := NewSupplyContext()
	healthy := ctx.Statuses().Healthy()

	// Call the function to ensure it's covered
	healthy.isSupplyStatusRef()
}

// TestTreatmentStatusRefChecker tests the isTreatmentStatusRef function
func TestTreatmentStatusRefChecker(_ *testing.T) {
	ctx := NewTreatmentContext()
	completed := ctx.Statuses().Completed()

	// Call the function to ensure it's covered
	completed.isTreatmentStatusRef()
}
