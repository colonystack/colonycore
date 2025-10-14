package datasetapi

import "testing"

func TestDatasetContextMarkerMethodsAreCallable(t *testing.T) {
	t.Run("SampleSourceRef", func(t *testing.T) {
		ref := NewSampleContext().Sources().Organism()
		ref.isSampleSourceRef()
		if ref.Equals(&sampleSourceRef{value: datasetSampleSourceOrganism}) {
			t.Fatal("sample source equality should reject pointer types")
		}
	})

	t.Run("SampleStatusRef", func(t *testing.T) {
		ref := NewSampleContext().Statuses().Stored()
		ref.isSampleStatusRef()
		if ref.Equals(&sampleStatusRef{value: datasetSampleStatusStored}) {
			t.Fatal("sample status equality should reject pointer types")
		}
	})

	t.Run("TreatmentStatusRef", func(t *testing.T) {
		ref := NewTreatmentContext().Statuses().Completed()
		ref.isTreatmentStatusRef()
		if ref.Equals(&treatmentStatusRef{value: datasetTreatmentStatusCompleted}) {
			t.Fatal("treatment status equality should reject pointer types")
		}
	})

	t.Run("SupplyStatusRef", func(t *testing.T) {
		ref := NewSupplyContext().Statuses().Critical()
		ref.isSupplyStatusRef()
		if ref.Equals(&supplyStatusRef{value: datasetSupplyStatusCritical}) {
			t.Fatal("supply status equality should reject pointer types")
		}
	})

	t.Run("PermitStatusRef", func(t *testing.T) {
		ref := NewPermitContext().Statuses().Active()
		ref.isPermitStatusRef()
		if ref.Equals(&permitStatusRef{value: datasetPermitStatusActive}) {
			t.Fatal("permit status equality should reject pointer types")
		}
	})

	t.Run("ProtocolStatusRef", func(t *testing.T) {
		ref := NewProtocolContext().Completed()
		ref.isProtocolStatusRef()
		if ref.Equals(&protocolStatusRef{value: "completed"}) {
			t.Fatal("protocol status equality should reject pointer types")
		}
	})

	t.Run("ProcedureStatusRef", func(t *testing.T) {
		ref := NewProcedureContext().InProgress()
		ref.isProcedureStatusRef()
		if ref.Equals(&procedureStatusRef{value: "in_progress"}) {
			t.Fatal("procedure status equality should reject pointer types")
		}
	})

	t.Run("CohortPurposeRef", func(t *testing.T) {
		ref := NewCohortContext().Research()
		ref.isCohortPurposeRef()
		if ref.Equals(&cohortPurposeRef{value: purposeResearch}) {
			t.Fatal("cohort purpose equality should reject pointer types")
		}
	})

	t.Run("BreedingStrategyRef", func(t *testing.T) {
		ref := NewBreedingContext().Artificial()
		ref.isBreedingStrategyRef()
		if ref.Equals(&breedingStrategyRef{value: strategyArtificial}) {
			t.Fatal("breeding strategy equality should reject pointer types")
		}
	})

	t.Run("ObservationShapeRef", func(t *testing.T) {
		ref := NewObservationContext().Shapes().Mixed()
		ref.isObservationShapeRef()
		if ref.Equals(&observationShapeRef{value: datasetObservationShapeMixed}) {
			t.Fatal("observation shape equality should reject pointer types")
		}
	})

	t.Run("HousingEnvironmentTypeRef", func(t *testing.T) {
		ref := NewHousingContext().Aquatic()
		ref.isEnvironmentTypeRef()
		if ref.Equals(&environmentTypeRef{value: envAquatic}) {
			t.Fatal("environment type equality should reject pointer types")
		}
	})

	t.Run("LifecycleStageRef", func(t *testing.T) {
		ref := NewLifecycleStageContext().Adult()
		ref.isLifecycleStageRef()
		if ref.Equals(&lifecycleStageRef{value: stageAdult}) {
			t.Fatal("lifecycle stage equality should reject pointer types")
		}
	})

	t.Run("FacilityZoneRef", func(t *testing.T) {
		ref := NewFacilityContext().Zones().Biosecure()
		ref.isFacilityZoneRef()
		if ref.Equals(&facilityZoneRef{value: facilityZoneBiosecure}) {
			t.Fatal("facility zone equality should reject pointer types")
		}
	})

	t.Run("FacilityAccessPolicyRef", func(t *testing.T) {
		ref := NewFacilityContext().AccessPolicies().Open()
		ref.isFacilityAccessPolicyRef()
		if ref.Equals(&facilityAccessPolicyRef{value: facilityAccessOpen}) {
			t.Fatal("facility access equality should reject pointer types")
		}
	})
}
