package datasetapi

import (
	"testing"
	"time"
)

const (
	testStatusDraft     = "draft"
	testStatusApproved  = "approved"
	testStatusCompleted = "completed"
	testStatusScheduled = "scheduled"
	testProtocolID      = "protocol1"
)

func TestOrganismContextualAccessors(t *testing.T) {
	t.Run("GetCurrentStage returns contextual stage reference", func(t *testing.T) {
		organism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org1"},
			Name:  "Test Organism",
			Stage: "adult",
		})

		stageRef := organism.GetCurrentStage()
		if stageRef.String() != "adult" {
			t.Errorf("Expected stage 'adult', got '%s'", stageRef.String())
		}
		if !stageRef.IsActive() {
			t.Error("Adult stage should be active")
		}
	})

	t.Run("IsActive returns correct lifecycle state", func(t *testing.T) {
		activeOrganism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org1"},
			Stage: "adult",
		})
		if !activeOrganism.IsActive() {
			t.Error("Adult organism should be active")
		}

		inactiveOrganism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org2"},
			Stage: "deceased",
		})
		if inactiveOrganism.IsActive() {
			t.Error("Deceased organism should not be active")
		}
	})

	t.Run("IsRetired returns correct retirement state", func(t *testing.T) {
		retiredOrganism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org1"},
			Stage: "retired",
		})
		if !retiredOrganism.IsRetired() {
			t.Error("Retired organism should return true for IsRetired")
		}

		activeOrganism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org2"},
			Stage: "adult",
		})
		if activeOrganism.IsRetired() {
			t.Error("Adult organism should return false for IsRetired")
		}
	})

	t.Run("IsDeceased returns correct death state", func(t *testing.T) {
		deceasedOrganism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org1"},
			Stage: "deceased",
		})
		if !deceasedOrganism.IsDeceased() {
			t.Error("Deceased organism should return true for IsDeceased")
		}

		aliveOrganism := NewOrganism(OrganismData{
			Base:  BaseData{ID: "org2"},
			Stage: "adult",
		})
		if aliveOrganism.IsDeceased() {
			t.Error("Adult organism should return false for IsDeceased")
		}
	})
}

func TestHousingUnitContextualAccessors(t *testing.T) {
	t.Run("GetEnvironmentType returns contextual environment reference", func(t *testing.T) {
		housing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing1"},
			Name:        "Test Tank",
			Environment: "aquatic",
			Capacity:    10,
		})

		envRef := housing.GetEnvironmentType()
		if envRef.String() != "aquatic" {
			t.Errorf("Expected environment 'aquatic', got '%s'", envRef.String())
		}
		if !envRef.IsAquatic() {
			t.Error("Aquatic environment should return true for IsAquatic")
		}
	})

	t.Run("IsAquaticEnvironment returns correct aquatic state", func(t *testing.T) {
		aquaticHousing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing1"},
			Environment: "aquatic",
		})
		if !aquaticHousing.IsAquaticEnvironment() {
			t.Error("Aquatic housing should return true for IsAquaticEnvironment")
		}

		terrestrialHousing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing2"},
			Environment: "terrestrial",
		})
		if terrestrialHousing.IsAquaticEnvironment() {
			t.Error("Terrestrial housing should return false for IsAquaticEnvironment")
		}
	})

	t.Run("IsHumidEnvironment returns correct humidity state", func(t *testing.T) {
		humidHousing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing1"},
			Environment: "humid",
		})
		if !humidHousing.IsHumidEnvironment() {
			t.Error("Humid housing should return true for IsHumidEnvironment")
		}

		dryHousing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing2"},
			Environment: "terrestrial",
		})
		if dryHousing.IsHumidEnvironment() {
			t.Error("Terrestrial housing should return false for IsHumidEnvironment")
		}
	})

	t.Run("SupportsSpecies returns correct species compatibility", func(t *testing.T) {
		aquaticHousing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing1"},
			Environment: "aquatic",
		})
		if !aquaticHousing.SupportsSpecies("fish") {
			t.Error("Aquatic housing should support fish")
		}
		if aquaticHousing.SupportsSpecies("bird") {
			t.Error("Aquatic housing should not support bird")
		}

		arborealHousing := NewHousingUnit(HousingUnitData{
			Base:        BaseData{ID: "housing2"},
			Environment: "arboreal",
		})
		if !arborealHousing.SupportsSpecies("bird") {
			t.Error("Arboreal housing should support bird")
		}
	})

	t.Run("Name and FacilityID accessors work", func(t *testing.T) {
		housing := NewHousingUnit(HousingUnitData{
			Base:       BaseData{ID: "housing1"},
			Name:       "Test Tank",
			FacilityID: "Lab A",
		})

		if housing.Name() != "Test Tank" {
			t.Errorf("Expected name 'Test Tank', got '%s'", housing.Name())
		}
		if housing.FacilityID() != "Lab A" {
			t.Errorf("Expected facility ID 'Lab A', got '%s'", housing.FacilityID())
		}
	})
}

func TestProtocolContextualAccessors(t *testing.T) {
	t.Run("GetCurrentStatus returns contextual status reference", func(t *testing.T) {
		protocol := NewProtocol(ProtocolData{
			Base:        BaseData{ID: "protocol1"},
			Code:        "P001",
			Title:       "Test Protocol",
			Description: strPtr("Test description"),
			MaxSubjects: 10,
		})

		// Default status should be "draft"
		statusRef := protocol.GetCurrentStatus()
		if statusRef.String() != testStatusDraft {
			t.Errorf("Expected status 'draft', got '%s'", statusRef.String())
		}
	})

	t.Run("IsActiveProtocol returns correct active state", func(t *testing.T) {
		activeProtocol := NewProtocol(ProtocolData{
			Base: BaseData{ID: "protocol1"},
			Code: "P001",
		})
		// Set status through internal field manipulation for testing
		// In real usage, status would be set through service operations
		p := activeProtocol.(protocol)
		p.status = testStatusApproved
		activeProtocol = p

		if !activeProtocol.IsActiveProtocol() {
			t.Error("Active protocol should return true for IsActiveProtocol")
		}
	})

	t.Run("IsTerminalStatus returns correct terminal state", func(t *testing.T) {
		completedProtocol := NewProtocol(ProtocolData{
			Base: BaseData{ID: "protocol1"},
			Code: "P001",
		})
		// Set status through internal field manipulation for testing
		p := completedProtocol.(protocol)
		p.status = datasetProtocolStatusExpired
		completedProtocol = p

		if !completedProtocol.IsTerminalStatus() {
			t.Error("Expired protocol should return true for IsTerminalStatus")
		}
	})

	t.Run("CanAcceptNewSubjects returns correct capacity state", func(t *testing.T) {
		availableProtocol := NewProtocol(ProtocolData{
			Base:        BaseData{ID: "protocol1"},
			Code:        "P001",
			MaxSubjects: 10,
		})
		// Set as active status
		p := availableProtocol.(protocol)
		p.status = testStatusApproved
		availableProtocol = p

		if !availableProtocol.CanAcceptNewSubjects() {
			t.Error("Approved protocol with capacity should accept new subjects")
		}

		// Test completed protocol
		p.status = testStatusCompleted
		availableProtocol = p
		if availableProtocol.CanAcceptNewSubjects() {
			t.Error("Completed protocol should not accept new subjects")
		}
	})

	t.Run("Description accessor works", func(t *testing.T) {
		protocol := NewProtocol(ProtocolData{
			Base:        BaseData{ID: "protocol1"},
			Description: strPtr("Test description"),
		})

		if protocol.Description() != "Test description" {
			t.Errorf("Expected description 'Test description', got '%s'", protocol.Description())
		}
	})
}

func TestProcedureContextualAccessors(t *testing.T) {
	t.Run("GetCurrentStatus returns contextual status reference", func(t *testing.T) {
		procedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc1"},
			Name:       "Test Procedure",
			Status:     testStatusScheduled,
			ProtocolID: testProtocolID,
		})

		statusRef := procedure.GetCurrentStatus()
		if statusRef.String() != testStatusScheduled {
			t.Errorf("Expected status 'scheduled', got '%s'", statusRef.String())
		}
	})

	t.Run("IsActiveProcedure returns correct active state", func(t *testing.T) {
		activeProcedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc1"},
			Status:     "in_progress",
			ProtocolID: testProtocolID,
		})

		if !activeProcedure.IsActiveProcedure() {
			t.Error("In-progress procedure should return true for IsActiveProcedure")
		}

		scheduledProcedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc2"},
			Status:     testStatusScheduled,
			ProtocolID: testProtocolID,
		})

		if scheduledProcedure.IsActiveProcedure() {
			t.Error("Scheduled procedure should return false for IsActiveProcedure")
		}
	})

	t.Run("IsTerminalStatus returns correct terminal state", func(t *testing.T) {
		completedProcedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc1"},
			Status:     testStatusCompleted,
			ProtocolID: testProtocolID,
		})

		if !completedProcedure.IsTerminalStatus() {
			t.Error("Completed procedure should return true for IsTerminalStatus")
		}

		activeProcedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc2"},
			Status:     "in_progress",
			ProtocolID: testProtocolID,
		})

		if activeProcedure.IsTerminalStatus() {
			t.Error("In-progress procedure should return false for IsTerminalStatus")
		}
	})

	t.Run("IsSuccessful returns correct success state", func(t *testing.T) {
		successfulProcedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc1"},
			Status:     testStatusCompleted,
			ProtocolID: testProtocolID,
		})

		if !successfulProcedure.IsSuccessful() {
			t.Error("Completed procedure should return true for IsSuccessful")
		}

		failedProcedure := NewProcedure(ProcedureData{
			Base:       BaseData{ID: "proc2"},
			Status:     "failed",
			ProtocolID: testProtocolID,
		})

		if failedProcedure.IsSuccessful() {
			t.Error("Failed procedure should return false for IsSuccessful")
		}
	})

	t.Run("Name, Status, ScheduledAt, ProtocolID accessors work", func(t *testing.T) {
		scheduledAt := time.Now()
		procedure := NewProcedure(ProcedureData{
			Base:        BaseData{ID: "proc1"},
			Name:        "Test Procedure",
			Status:      testStatusScheduled,
			ScheduledAt: scheduledAt,
			ProtocolID:  testProtocolID,
		})

		if procedure.Name() != "Test Procedure" {
			t.Errorf("Expected name 'Test Procedure', got '%s'", procedure.Name())
		}
		if procedure.Status() != testStatusScheduled {
			t.Errorf("Expected status 'scheduled', got '%s'", procedure.Status())
		}
		if !procedure.ScheduledAt().Equal(scheduledAt) {
			t.Errorf("Expected scheduled at %v, got %v", scheduledAt, procedure.ScheduledAt())
		}
		if procedure.ProtocolID() != testProtocolID {
			t.Errorf("Expected protocol ID 'protocol1', got '%s'", procedure.ProtocolID())
		}
	})
}

func TestCohortAccessors(t *testing.T) {
	t.Run("HousingID and ProtocolID accessors work", func(t *testing.T) {
		housingID := "housing1"
		protocolID := testProtocolID
		cohort := NewCohort(CohortData{
			Base:       BaseData{ID: "cohort1"},
			Name:       "Test Cohort",
			HousingID:  &housingID,
			ProtocolID: &protocolID,
		})

		housing, hasHousing := cohort.HousingID()
		if !hasHousing {
			t.Error("Expected cohort to have housing ID")
		}
		if housing != "housing1" {
			t.Errorf("Expected housing ID 'housing1', got '%s'", housing)
		}

		protocol, hasProtocol := cohort.ProtocolID()
		if !hasProtocol {
			t.Error("Expected cohort to have protocol ID")
		}
		if protocol != testProtocolID {
			t.Errorf("Expected protocol ID 'protocol1', got '%s'", protocol)
		}
	})

	t.Run("contextual purpose accessors work", func(t *testing.T) {
		cohort := NewCohort(CohortData{
			Base:    BaseData{ID: "cohort1"},
			Name:    "Research Cohort",
			Purpose: "research",
		})

		purposeRef := cohort.GetPurpose()
		if purposeRef.String() != purposeResearch {
			t.Errorf("Expected purpose '%s', got '%s'", purposeResearch, purposeRef.String())
		}

		if !cohort.IsResearchCohort() {
			t.Error("Research cohort should return true for IsResearchCohort()")
		}

		if !cohort.RequiresProtocol() {
			t.Error("Research cohort should require protocol")
		}

		// Test breeding cohort
		breedingCohort := NewCohort(CohortData{
			Base:    BaseData{ID: "cohort2"},
			Purpose: "breeding",
		})

		if breedingCohort.IsResearchCohort() {
			t.Error("Breeding cohort should return false for IsResearchCohort()")
		}
	})
}

func TestBreedingUnitContextualAccessors(t *testing.T) {
	t.Run("contextual strategy accessors work", func(t *testing.T) {
		breedingUnit := NewBreedingUnit(BreedingUnitData{
			Base:     BaseData{ID: "breeding1"},
			Name:     "Natural Breeding Unit",
			Strategy: strategyNatural,
		})

		strategyRef := breedingUnit.GetBreedingStrategy()
		if strategyRef.String() != strategyNatural {
			t.Errorf("Expected strategy '%s', got '%s'", strategyNatural, strategyRef.String())
		}

		if !breedingUnit.IsNaturalBreeding() {
			t.Error("Natural breeding unit should return true for IsNaturalBreeding()")
		}

		if breedingUnit.RequiresIntervention() {
			t.Error("Natural breeding should not require intervention")
		}

		// Test artificial breeding
		artificialUnit := NewBreedingUnit(BreedingUnitData{
			Base:     BaseData{ID: "breeding2"},
			Strategy: "artificial",
		})

		if artificialUnit.IsNaturalBreeding() {
			t.Error("Artificial breeding unit should return false for IsNaturalBreeding()")
		}

		if !artificialUnit.RequiresIntervention() {
			t.Error("Artificial breeding should require intervention")
		}
	})

	t.Run("Strategy and ProtocolID accessors work", func(t *testing.T) {
		protocolID := testProtocolID
		breeding := NewBreedingUnit(BreedingUnitData{
			Base:       BaseData{ID: "breeding1"},
			Name:       "Test Breeding",
			Strategy:   "natural",
			ProtocolID: &protocolID,
		})

		if breeding.Strategy() != strategyNatural {
			t.Errorf("Expected strategy '%s', got '%s'", strategyNatural, breeding.Strategy())
		}

		protocol, hasProtocol := breeding.ProtocolID()
		if !hasProtocol {
			t.Error("Expected breeding unit to have protocol ID")
		}
		if protocol != testProtocolID {
			t.Errorf("Expected protocol ID 'protocol1', got '%s'", protocol)
		}
	})
}
