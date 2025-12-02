package pluginapi

import "testing"

func TestContextMarkerMethodsAreCallable(t *testing.T) {
	t.Run("ActionRef", func(t *testing.T) {
		ref := NewActionContext().Create()
		ref.isActionRef()
		if ref.Equals(&actionRef{value: actionCreate}) {
			t.Fatal("action equality should reject pointer types")
		}
	})

	t.Run("EntityTypeRef", func(t *testing.T) {
		ref := NewEntityContext().Organism()
		ref.isEntityTypeRef()
		if ref.Equals(&entityTypeRef{value: entityOrganism}) {
			t.Fatal("entity equality should reject pointer types")
		}
	})

	t.Run("LifecycleStageRef", func(t *testing.T) {
		ref := NewLifecycleStageContext().Adult()
		ref.isLifecycleStageRef()
		if ref.Equals(&lifecycleStageRef{value: stageAdult}) {
			t.Fatal("lifecycle equality should reject pointer types")
		}
	})

	t.Run("SampleSourceRef", func(t *testing.T) {
		ref := NewSampleContext().Sources().Organism()
		ref.isSampleSourceRef()
		if ref.Equals(&sampleSourceRef{value: sampleSourceOrganism}) {
			t.Fatal("sample source equality should reject pointer types")
		}
	})

	t.Run("SampleStatusRef", func(t *testing.T) {
		ref := NewSampleContext().Statuses().Stored()
		ref.isSampleStatusRef()
		if ref.Equals(&sampleStatusRef{value: sampleStatusStored}) {
			t.Fatal("sample status equality should reject pointer types")
		}
	})

	t.Run("SeverityRef", func(t *testing.T) {
		ref := NewSeverityContext().Warn()
		ref.isSeverityRef()
		if ref.Equals(&severityRef{value: severityWarn}) {
			t.Fatal("severity equality should reject pointer types")
		}
	})

	t.Run("SupplyStatusRef", func(t *testing.T) {
		ref := NewSupplyContext().Statuses().Reorder()
		ref.isSupplyStatusRef()
		if ref.Equals(&supplyStatusRef{value: supplyStatusReorder}) {
			t.Fatal("supply status equality should reject pointer types")
		}
	})

	t.Run("TreatmentStatusRef", func(t *testing.T) {
		ref := NewTreatmentContext().Statuses().Completed()
		ref.isTreatmentStatusRef()
		if ref.Equals(&treatmentStatusRef{value: treatmentStatusCompleted}) {
			t.Fatal("treatment status equality should reject pointer types")
		}
	})

	t.Run("FacilityZoneRef", func(t *testing.T) {
		ref := NewFacilityContext().Zones().Biosecure()
		ref.isFacilityZoneRef()
		if ref.Equals(&facilityZoneRef{value: zoneBiosecure}) {
			t.Fatal("facility zone equality should reject pointer types")
		}
	})

	t.Run("FacilityAccessPolicyRef", func(t *testing.T) {
		ref := NewFacilityContext().AccessPolicies().Restricted()
		ref.isFacilityAccessPolicyRef()
		if ref.Equals(&facilityAccessPolicyRef{value: accessRestricted}) {
			t.Fatal("facility access equality should reject pointer types")
		}
	})

	t.Run("ObservationShapeRef", func(t *testing.T) {
		ref := NewObservationContext().Shapes().Structured()
		ref.isObservationShapeRef()
		if ref.Equals(&observationShapeRef{value: observationShapeStructured}) {
			t.Fatal("observation shape equality should reject pointer types")
		}
	})

	t.Run("HousingEnvironmentTypeRef", func(t *testing.T) {
		ref := NewHousingContext().Aquatic()
		ref.isEnvironmentTypeRef()
		if ref.Equals(&environmentTypeRef{value: "aquatic"}) {
			t.Fatal("environment equality should reject pointer types")
		}
	})

	t.Run("HousingStateRef", func(t *testing.T) {
		ref := NewHousingStateContext().Active()
		ref.isHousingStateRef()
		if ref.Equals(&housingStateRef{value: housingStateActive}) {
			t.Fatal("housing state equality should reject pointer types")
		}
	})

	t.Run("ProtocolStatusRef", func(t *testing.T) {
		ref := NewProtocolContext().Draft()
		ref.isProtocolStatusRef()
		if ref.Equals(&protocolStatusRef{value: "draft"}) {
			t.Fatal("protocol status equality should reject pointer types")
		}
	})

	t.Run("PermitStatusRef", func(t *testing.T) {
		ref := NewPermitContext().Statuses().Approved()
		ref.isPermitStatusRef()
		if ref.Equals(&permitStatusRef{value: permitStatusApproved}) {
			t.Fatal("permit status equality should reject pointer types")
		}
	})
}
