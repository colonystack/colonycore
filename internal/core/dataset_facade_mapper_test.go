package core

import (
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

const (
	testMapperEnvAquatic     = "aquatic"
	testMapperStatusApproved = "approved"
	testMapperAdultStage     = "adult"
	testMapperFacilityID     = "facility-1"
)

func TestFacadeHousingUnitFromDomain(t *testing.T) {
	now := time.Now()
	unit := domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "housing-1",

		CreatedAt: now,

		UpdatedAt:   now,
		Name:        "Quarantine Tank",
		FacilityID:  testMapperFacilityID,
		Capacity:    4,
		Environment: domain.HousingEnvironmentAquatic,
		State:       domain.HousingStateQuarantine},
	}

	facade := facadeHousingUnitFromDomain(unit)
	if facade.State() != "quarantine" {
		t.Fatalf("expected state to map through, got %s", facade.State())
	}
	if !facade.GetState().Equals(datasetapi.NewHousingStateContext().Quarantine()) {
		t.Fatal("expected quarantine state reference to map through")
	}
	if facade.Environment() != testMapperEnvAquatic || facade.Capacity() != 4 || facade.FacilityID() != testMapperFacilityID {
		t.Fatalf("unexpected housing mapping: %+v", facade)
	}
}

func TestFacadeHousingUnitsFromDomainEmpty(t *testing.T) {
	if out := facadeHousingUnitsFromDomain(nil); out != nil {
		t.Fatalf("expected nil for nil input, got %+v", out)
	}
	if out := facadeHousingUnitsFromDomain([]domain.HousingUnit{}); out != nil {
		t.Fatalf("expected nil for empty slice input, got %+v", out)
	}
}

func TestFacadeProtocolMapping(t *testing.T) {
	now := time.Now()
	protocol := domain.Protocol{Protocol: entitymodel.Protocol{ID: "protocol-1",

		CreatedAt: now,

		UpdatedAt:   now,
		Code:        "P1",
		Title:       "Title",
		Description: strPtr("desc"),
		MaxSubjects: 5,
		Status:      domain.ProtocolStatusApproved},
	}

	mapped := facadeProtocolFromDomain(protocol)
	if mapped.GetCurrentStatus().String() != testMapperStatusApproved || mapped.Code() != "P1" {
		t.Fatalf("unexpected protocol mapping")
	}

	// Ensure slice handling in plural helper
	list := facadeProtocolsFromDomain([]domain.Protocol{protocol})
	if len(list) != 1 || list[0].Code() != "P1" {
		t.Fatalf("expected single protocol mapping, got %+v", list)
	}
	if facadeProtocolsFromDomain(nil) != nil || facadeProtocolsFromDomain([]domain.Protocol{}) != nil {
		t.Fatal("expected nil slices for empty protocol inputs")
	}
}

func TestFacadeProjectMappingClonesSlices(t *testing.T) {
	project := domain.Project{Project: entitymodel.Project{ID: "project-1",
		Code:         "PRJ",
		Title:        "Project",
		Description:  strPtr("desc"),
		FacilityIDs:  []string{testMapperFacilityID},
		ProtocolIDs:  []string{"protocol-1"},
		OrganismIDs:  []string{"organism-1"},
		ProcedureIDs: []string{"procedure-1"},
		SupplyItemIDs: []string{
			"supply-1",
		}},
	}

	mapped := facadeProjectFromDomain(project)
	if mapped.Code() != "PRJ" || mapped.Title() != "Project" {
		t.Fatalf("unexpected project mapping: %+v", mapped)
	}
	// mutate originals to ensure cloning
	project.FacilityIDs[0] = "mutated"
	if mapped.FacilityIDs()[0] != testMapperFacilityID {
		t.Fatal("project facilities should be cloned")
	}

	if facadeProjectsFromDomain(nil) != nil || facadeProjectsFromDomain([]domain.Project{}) != nil {
		t.Fatal("expected nil slices for empty project inputs")
	}
}

func TestFacadeCohortAndTreatmentMapping(t *testing.T) {
	cohort := domain.Cohort{Cohort: entitymodel.Cohort{ID: "cohort-1",
		Name:      "Cohort",
		Purpose:   "Research",
		ProjectID: strPtr("project-1"),
		HousingID: strPtr("housing-1")},
	}
	mappedCohort := facadeCohortFromDomain(cohort)
	if mappedCohort.Name() != "Cohort" || mappedCohort.Purpose() != "Research" {
		t.Fatalf("unexpected cohort mapping: %+v", mappedCohort)
	}
	if facadeCohortsFromDomain(nil) != nil || facadeCohortsFromDomain([]domain.Cohort{}) != nil {
		t.Fatal("expected nil slices for empty cohort inputs")
	}

	treatment := domain.Treatment{Treatment: entitymodel.Treatment{ID: "treatment-1",
		Name:              "Treatment",
		Status:            domain.TreatmentStatusCompleted,
		ProcedureID:       "procedure-1",
		OrganismIDs:       []string{"organism-1"},
		CohortIDs:         []string{"cohort-1"},
		DosagePlan:        "plan",
		AdministrationLog: []string{"log"},
		AdverseEvents:     nil},
	}
	mappedTreatment := facadeTreatmentFromDomain(treatment)
	if mappedTreatment.Name() != "Treatment" || mappedTreatment.GetCurrentStatus().String() != "completed" {
		t.Fatalf("unexpected treatment mapping")
	}
	if facadeTreatmentsFromDomain(nil) != nil || facadeTreatmentsFromDomain([]domain.Treatment{}) != nil {
		t.Fatal("expected nil slices for empty treatment inputs")
	}
}

func TestFacadeOrganismMapping(t *testing.T) {
	org := domain.Organism{Organism: entitymodel.Organism{ID: "org-1",
		Name:      "Org",
		Species:   "species",
		Line:      "line",
		Stage:     domain.StageAdult,
		ParentIDs: []string{"parent-1"}},
	}
	if err := org.SetCoreAttributes(map[string]any{"flag": true}); err != nil {
		t.Fatalf("set core attributes: %v", err)
	}
	mapped := facadeOrganismFromDomain(org)
	if mapped.Name() != "Org" || mapped.Species() != "species" || mapped.GetCurrentStage().String() != testMapperAdultStage {
		t.Fatalf("unexpected organism mapping: %+v", mapped)
	}
	hook := datasetapi.NewExtensionHookContext().OrganismAttributes()
	payload, ok := mapped.Extensions().Core(hook)
	if !ok || payload.Map()["flag"] != true {
		t.Fatalf("expected core extension payload mapping, got %+v", payload)
	}
	if facadeOrganismsFromDomain(nil) != nil || facadeOrganismsFromDomain([]domain.Organism{}) != nil {
		t.Fatal("expected nil slices for empty organism inputs")
	}
}

func TestFacadeBreedingAndProcedureMapping(t *testing.T) {
	now := time.Now()
	breeding := domain.BreedingUnit{BreedingUnit: entitymodel.BreedingUnit{ID: "breeding-1",
		Name:          "Breeding",
		Strategy:      "natural",
		FemaleIDs:     []string{"f1"},
		MaleIDs:       []string{"m1"},
		PairingIntent: strPtr("pair"),
		PairingNotes:  strPtr("notes")},
	}
	mappedBreeding := facadeBreedingUnitFromDomain(breeding)
	if mappedBreeding.Name() != "Breeding" || len(mappedBreeding.FemaleIDs()) != 1 {
		t.Fatalf("unexpected breeding mapping: %+v", mappedBreeding)
	}
	if facadeBreedingUnitsFromDomain(nil) != nil || facadeBreedingUnitsFromDomain([]domain.BreedingUnit{}) != nil {
		t.Fatal("expected nil slices for empty breeding inputs")
	}

	procedure := domain.Procedure{Procedure: entitymodel.Procedure{ID: "procedure-1",
		Name:        "Procedure",
		Status:      domain.ProcedureStatusScheduled,
		ScheduledAt: now,
		ProtocolID:  "protocol-1",
		OrganismIDs: []string{"org-1"}},
	}
	mappedProcedure := facadeProcedureFromDomain(procedure)
	if mappedProcedure.Name() != "Procedure" || mappedProcedure.GetCurrentStatus().String() != "scheduled" {
		t.Fatalf("unexpected procedure mapping: %+v", mappedProcedure)
	}
	if facadeProceduresFromDomain(nil) != nil || facadeProceduresFromDomain([]domain.Procedure{}) != nil {
		t.Fatal("expected nil slices for empty procedure inputs")
	}
}

func TestFacadeFacilityPermitSampleSupplyMapping(t *testing.T) {
	now := time.Now()
	facility := domain.Facility{Facility: entitymodel.Facility{ID: testMapperFacilityID,
		Code:           "FAC",
		Name:           "Facility",
		Zone:           "ZoneA",
		AccessPolicy:   "restricted",
		HousingUnitIDs: []string{"housing-1"},
		ProjectIDs:     []string{"project-1"}},
	}
	facilityFacade := facadeFacilityFromDomain(facility)
	if facilityFacade.Code() != "FAC" || facilityFacade.AccessPolicy() != "restricted" {
		t.Fatalf("unexpected facility mapping: %+v", facilityFacade)
	}
	if facadeFacilitiesFromDomain(nil) != nil || facadeFacilitiesFromDomain([]domain.Facility{}) != nil {
		t.Fatal("expected nil slices for empty facility inputs")
	}

	permit := domain.Permit{Permit: entitymodel.Permit{ID: "permit-1",
		PermitNumber:      "P1",
		Authority:         "Auth",
		Status:            domain.PermitStatusApproved,
		ValidFrom:         now.Add(-time.Hour),
		ValidUntil:        now.Add(time.Hour),
		AllowedActivities: []string{"Activity"},
		FacilityIDs:       []string{testMapperFacilityID},
		ProtocolIDs:       []string{"protocol-1"},
		Notes:             strPtr("note")},
	}
	permitFacade := facadePermitFromDomain(permit)
	if permitFacade.GetStatus(now).String() != testMapperStatusApproved || !permitFacade.IsActive(now) {
		t.Fatalf("unexpected permit mapping: %+v", permitFacade)
	}
	if facadePermitsFromDomain(nil) != nil || facadePermitsFromDomain([]domain.Permit{}) != nil {
		t.Fatal("expected nil slices for empty permit inputs")
	}

	sample := domain.Sample{Sample: entitymodel.Sample{ID: "sample-1",
		Identifier:      "S1",
		SourceType:      "organism",
		FacilityID:      testMapperFacilityID,
		CollectedAt:     now,
		Status:          domain.SampleStatusStored,
		StorageLocation: "Freezer",
		AssayType:       "assay",
		ChainOfCustody: []domain.SampleCustodyEvent{
			{Actor: "tech", Location: "lab", Timestamp: now},
		}},
	}
	sampleFacade := facadeSampleFromDomain(sample)
	if sampleFacade.Status() != "stored" || len(sampleFacade.ChainOfCustody()) != 1 {
		t.Fatalf("unexpected sample mapping: %+v", sampleFacade)
	}
	if facadeSamplesFromDomain(nil) != nil || facadeSamplesFromDomain([]domain.Sample{}) != nil {
		t.Fatal("expected nil slices for empty sample inputs")
	}

	supply := domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{ID: "supply-1",
		SKU:            "SKU",
		Name:           "Supply",
		QuantityOnHand: 1,
		Unit:           "unit",
		FacilityIDs:    []string{testMapperFacilityID},
		ProjectIDs:     []string{"project-1"},
		ReorderLevel:   2},
	}
	supplyFacade := facadeSupplyItemFromDomain(supply)
	if supplyFacade.SKU() != "SKU" || supplyFacade.Unit() != "unit" {
		t.Fatalf("unexpected supply mapping: %+v", supplyFacade)
	}
	if facadeSupplyItemsFromDomain(nil) != nil || facadeSupplyItemsFromDomain([]domain.SupplyItem{}) != nil {
		t.Fatal("expected nil slices for empty supply inputs")
	}
}
