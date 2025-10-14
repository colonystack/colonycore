package datasetapi

import (
	"testing"
	"time"
)

// TestBreedingContextRefChecker tests the isBreedingStrategyRef function
func TestBreedingContextRefChecker(_ *testing.T) {
	ctx := NewBreedingContext()
	strategy := ctx.Artificial()
	// Call the function to ensure it's covered
	strategy.isBreedingStrategyRef()
}

// TestCohortContextRefChecker tests the isCohortPurposeRef function
func TestCohortContextRefChecker(_ *testing.T) {
	ctx := NewCohortContext()
	purpose := ctx.Research()
	// Call the function to ensure it's covered
	purpose.isCohortPurposeRef()
}

// TestFacilityZoneRefChecker tests the isFacilityZoneRef function
func TestFacilityZoneRefChecker(_ *testing.T) {
	ctx := NewFacilityContext()
	zone := ctx.Zones().Biosecure()
	// Call the function to ensure it's covered
	zone.isFacilityZoneRef()
}

// TestFacilityAccessPolicyRefChecker tests the isFacilityAccessPolicyRef function
func TestFacilityAccessPolicyRefChecker(_ *testing.T) {
	ctx := NewFacilityContext()
	policy := ctx.AccessPolicies().Open()
	// Call the function to ensure it's covered
	policy.isFacilityAccessPolicyRef()
}

// TestEnvironmentTypeRefChecker tests the isEnvironmentTypeRef function
func TestEnvironmentTypeRefChecker(_ *testing.T) {
	ctx := NewHousingContext()
	envType := ctx.Aquatic()
	// Call the function to ensure it's covered
	envType.isEnvironmentTypeRef()
}

// TestLifecycleStageRefChecker tests the isLifecycleStageRef function
func TestLifecycleStageRefChecker(_ *testing.T) {
	ctx := NewLifecycleStageContext()
	stage := ctx.Adult()
	// Call the function to ensure it's covered
	stage.isLifecycleStageRef()
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
	active := ctx.Statuses().Active()

	// Call the function to ensure it's covered
	active.isPermitStatusRef()
}

// TestProcedureStatusRefChecker tests the isProcedureStatusRef function
func TestProcedureStatusRefChecker(_ *testing.T) {
	ctx := NewProcedureContext()
	completed := ctx.Completed()

	// Call the function to ensure it's covered
	completed.isProcedureStatusRef()
}

// TestProtocolStatusRefChecker tests the isProtocolStatusRef function
func TestProtocolStatusRefChecker(_ *testing.T) {
	ctx := NewProtocolContext()
	active := ctx.Active()

	// Call the function to ensure it's covered
	active.isProtocolStatusRef()
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

// TestFacadeMethods tests facade constructors with 0% coverage
func TestFacadeMethods(t *testing.T) {
	baseData := BaseData{
		ID:        "test-id",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test NewTreatment with minimal data
	treatmentData := TreatmentData{
		Base:        baseData,
		Name:        "Test Treatment",
		ProcedureID: "proc-123",
	}
	treatment := NewTreatment(treatmentData)
	if treatment.Name() != "Test Treatment" {
		t.Error("NewTreatment should create treatment with correct name")
	}

	// Test NewObservation with minimal data
	observationData := ObservationData{
		Base:       baseData,
		RecordedAt: time.Now(),
		Observer:   "Test Observer",
	}
	observation := NewObservation(observationData)
	if observation.Observer() != "Test Observer" {
		t.Error("NewObservation should create observation with correct observer")
	}

	// Test NewSample with minimal data
	sampleData := SampleData{
		Base:            baseData,
		Identifier:      "sample-123",
		SourceType:      "blood",
		FacilityID:      "facility-1",
		CollectedAt:     time.Now(),
		Status:          "collected",
		StorageLocation: "freezer-A",
		AssayType:       "PCR",
	}
	sample := NewSample(sampleData)
	if sample.Identifier() != "sample-123" {
		t.Error("NewSample should create sample with correct identifier")
	}

	// Test NewPermit with minimal data
	permitData := PermitData{
		Base:         baseData,
		PermitNumber: "permit-456",
		Authority:    "Test Authority",
		ValidFrom:    time.Now(),
		ValidUntil:   time.Now().Add(365 * 24 * time.Hour),
	}
	permit := NewPermit(permitData)
	if permit.PermitNumber() != "permit-456" {
		t.Error("NewPermit should create permit with correct permit number")
	}
}

// TestDerefTime tests the derefTime function with 0% coverage
func TestDerefTime(t *testing.T) {
	testTime := time.Now()

	// Test with non-nil pointer
	result, ok := derefTime(&testTime)
	if !ok || result == nil || !result.Equal(testTime) {
		t.Errorf("Expected derefTime to return %v, got %v", testTime, result)
	}

	// Test with nil pointer
	zeroTime, ok2 := derefTime(nil)
	if ok2 || zeroTime != nil {
		t.Errorf("Expected derefTime to return nil for nil pointer, got %v", zeroTime)
	}
}
