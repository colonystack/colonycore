package core

import (
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

const (
	testAttributeKey           = "key"
	testAttributeOriginalValue = "value"
	testCohortID               = "cohort"
	testHousingID              = "housing"
	testProtocolID             = "protocol"
	testProjectID              = "project"
)

func TestFacadeOrganismFromDomainCopiesData(t *testing.T) {
	now := time.Now()
	cohort := testCohortID
	housing := testHousingID
	protocol := testProtocolID
	project := testProjectID
	org := domain.Organism{
		Base:       domain.Base{ID: "id", CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
		Name:       "Name",
		Species:    "Species",
		Line:       "Line",
		Stage:      domain.StageAdult,
		CohortID:   &cohort,
		HousingID:  &housing,
		ProtocolID: &protocol,
		ProjectID:  &project,
		Attributes: map[string]any{testAttributeKey: testAttributeOriginalValue},
	}

	converted := facadeOrganismFromDomain(org)

	if converted.ID() != org.ID || converted.Stage() != datasetapi.LifecycleStage(org.Stage) {
		t.Fatalf("unexpected conversion: %+v", converted)
	}

	attrs := converted.Attributes()
	attrs[testAttributeKey] = testLiteralMutated
	if org.Attributes[testAttributeKey] != testAttributeOriginalValue {
		t.Fatalf("original attributes mutated: %+v", org.Attributes)
	}
	if cohortID, ok := converted.CohortID(); !ok || cohortID != cohort {
		t.Fatalf("expected cohort id clone")
	}
	cohort = testLiteralMutated
	if idAfter, _ := converted.CohortID(); idAfter != "cohort" {
		t.Fatalf("expected cohort clone to remain stable, got %s", idAfter)
	}
	housing = testLiteralMutated
	if idAfter, _ := converted.HousingID(); idAfter == testLiteralMutated {
		t.Fatalf("expected housing clone to remain stable")
	}
	protocol = testLiteralMutated
	if idAfter, _ := converted.ProtocolID(); idAfter == testLiteralMutated {
		t.Fatalf("expected protocol clone to remain stable")
	}
	project = testLiteralMutated
	if idAfter, _ := converted.ProjectID(); idAfter == testLiteralMutated {
		t.Fatalf("expected project clone to remain stable")
	}
}

func TestFacadeCollectionsCloneSlices(t *testing.T) {
	now := time.Now()
	cohortID := testCohortID
	housing := domain.HousingUnit{Base: domain.Base{ID: "H"}, Name: "Hab", Facility: "F", Capacity: 3, Environment: "wet"}
	protocol := domain.Protocol{Base: domain.Base{ID: "P"}, Code: "C", Title: "T", Description: "D", MaxSubjects: 5}
	project := domain.Project{Base: domain.Base{ID: "PR"}, Code: "CC", Title: "Title", Description: "Desc"}
	cohort := domain.Cohort{Base: domain.Base{ID: "C"}, Name: "Group", Purpose: "Study", ProjectID: &cohortID, HousingID: &cohortID, ProtocolID: &cohortID}
	breeding := domain.BreedingUnit{Base: domain.Base{ID: "B"}, Name: "Breed", Strategy: "Pair", HousingID: &cohortID, ProtocolID: &cohortID, FemaleIDs: []string{"f1"}, MaleIDs: []string{"m1"}}
	procedure := domain.Procedure{Base: domain.Base{ID: "PROC", UpdatedAt: now}, Name: "Proc", Status: "pending", ScheduledAt: now.Add(time.Hour), ProtocolID: "P", CohortID: &cohortID, OrganismIDs: []string{"o1"}}

	cohorts := facadeCohortsFromDomain([]domain.Cohort{cohort})
	housingUnits := facadeHousingUnitsFromDomain([]domain.HousingUnit{housing})
	protocols := facadeProtocolsFromDomain([]domain.Protocol{protocol})
	projects := facadeProjectsFromDomain([]domain.Project{project})
	breedingUnits := facadeBreedingUnitsFromDomain([]domain.BreedingUnit{breeding})
	procedures := facadeProceduresFromDomain([]domain.Procedure{procedure})

	if len(cohorts) != 1 || len(housingUnits) != 1 || len(protocols) != 1 || len(projects) != 1 || len(breedingUnits) != 1 || len(procedures) != 1 {
		t.Fatalf("unexpected collection conversion counts")
	}

	females := breedingUnits[0].FemaleIDs()
	females[0] = testLiteralMutated
	if breeding.FemaleIDs[0] != "f1" {
		t.Fatalf("expected breeding slice clone")
	}
	procIDs := procedures[0].OrganismIDs()
	procIDs[0] = testLiteralMutated
	if procedure.OrganismIDs[0] != "o1" {
		t.Fatalf("expected procedure slice clone")
	}
}

func TestFacadeCollectionsNilBehavior(t *testing.T) {
	if got := facadeOrganismsFromDomain(nil); got != nil {
		t.Fatalf("expected nil organisms slice")
	}
	if got := facadeHousingUnitsFromDomain(nil); got != nil {
		t.Fatalf("expected nil housing slice")
	}
	if got := facadeProtocolsFromDomain(nil); got != nil {
		t.Fatalf("expected nil protocols slice")
	}
	if got := facadeProjectsFromDomain(nil); got != nil {
		t.Fatalf("expected nil projects slice")
	}
	if got := facadeCohortsFromDomain(nil); got != nil {
		t.Fatalf("expected nil cohorts slice")
	}
	if got := facadeBreedingUnitsFromDomain(nil); got != nil {
		t.Fatalf("expected nil breeding slice")
	}
	if got := facadeProceduresFromDomain(nil); got != nil {
		t.Fatalf("expected nil procedures slice")
	}
}
