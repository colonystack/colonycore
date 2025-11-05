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

func strPtr(v string) *string {
	return &v
}

func TestFacadeOrganismFromDomainCopiesData(t *testing.T) {
	now := time.Now()
	cohort := testCohortID
	housing := testHousingID
	protocol := testProtocolID
	project := testProjectID
	lineID := "line-id"
	strainID := "strain-id"
	parentIDs := []string{"p1", "p2"}
	org := domain.Organism{
		Base:       domain.Base{ID: "id", CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
		Name:       "Name",
		Species:    "Species",
		Line:       "Line",
		LineID:     &lineID,
		StrainID:   &strainID,
		ParentIDs:  append([]string(nil), parentIDs...),
		Stage:      domain.StageAdult,
		CohortID:   &cohort,
		HousingID:  &housing,
		ProtocolID: &protocol,
		ProjectID:  &project,
	}
	if err := org.SetCoreAttributes(map[string]any{testAttributeKey: testAttributeOriginalValue}); err != nil {
		t.Fatalf("SetCoreAttributes: %v", err)
	}

	converted := facadeOrganismFromDomain(org)

	if converted.ID() != org.ID || converted.Stage() != datasetapi.LifecycleStage(org.Stage) {
		t.Fatalf("unexpected conversion: %+v", converted)
	}
	if got, ok := converted.LineID(); !ok || got != lineID {
		t.Fatalf("expected line id clone")
	}
	if got, ok := converted.StrainID(); !ok || got != strainID {
		t.Fatalf("expected strain id clone")
	}
	clonedParents := converted.ParentIDs()
	if len(clonedParents) != len(parentIDs) || clonedParents[0] != "p1" {
		t.Fatalf("unexpected parent ids: %+v", clonedParents)
	}

	attrs := converted.Attributes()
	attrs[testAttributeKey] = testLiteralMutated
	if org.CoreAttributes()[testAttributeKey] != testAttributeOriginalValue {
		t.Fatalf("original attributes mutated: %+v", org.CoreAttributes())
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
	parentIDs[0] = testLiteralMutated
	if converted.ParentIDs()[0] != "p1" {
		t.Fatalf("expected parent id clone to remain stable")
	}
}

func TestFacadeCollectionsCloneSlices(t *testing.T) {
	now := time.Now()
	cohortID := testCohortID
	housing := domain.HousingUnit{Base: domain.Base{ID: "H"}, Name: "Hab", FacilityID: "F", Capacity: 3, Environment: "wet"}
	protocol := domain.Protocol{Base: domain.Base{ID: "P"}, Code: "C", Title: "T", Description: strPtr("D"), MaxSubjects: 5}
	project := domain.Project{
		Base:          domain.Base{ID: "PR"},
		Code:          "CC",
		Title:         "Title",
		Description:   strPtr("Desc"),
		FacilityIDs:   []string{"facility"},
		ProtocolIDs:   []string{"protocol"},
		OrganismIDs:   []string{"organism"},
		ProcedureIDs:  []string{"procedure"},
		SupplyItemIDs: []string{"supply"},
	}
	cohort := domain.Cohort{Base: domain.Base{ID: "C"}, Name: "Group", Purpose: "Study", ProjectID: &cohortID, HousingID: &cohortID, ProtocolID: &cohortID}
	lineID := "line-1"
	lineSnapshot := lineID
	strainID := "strain-1"
	strainSnapshot := strainID
	targetLineID := "line-2"
	targetLineSnapshot := targetLineID
	targetStrainID := "strain-2"
	targetStrainSnapshot := targetStrainID
	breeding := domain.BreedingUnit{
		Base:           domain.Base{ID: "B"},
		Name:           "Breed",
		Strategy:       "Pair",
		HousingID:      &cohortID,
		ProtocolID:     &cohortID,
		LineID:         &lineID,
		StrainID:       &strainID,
		TargetLineID:   &targetLineID,
		TargetStrainID: &targetStrainID,
		PairingIntent:  strPtr("outcross"),
		PairingNotes:   strPtr("Documented pairing"),
		FemaleIDs:      []string{"f1"},
		MaleIDs:        []string{"m1"},
	}
	breeding.SetPairingAttributes(map[string]any{"purpose": "lineage"})
	procProject := testProjectID
	procedure := domain.Procedure{
		Base:           domain.Base{ID: "PROC", UpdatedAt: now},
		Name:           "Proc",
		Status:         "pending",
		ScheduledAt:    now.Add(time.Hour),
		ProtocolID:     "P",
		ProjectID:      &procProject,
		CohortID:       &cohortID,
		OrganismIDs:    []string{"o1"},
		TreatmentIDs:   []string{"t1"},
		ObservationIDs: []string{"obs1"},
	}

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
	if got, ok := breedingUnits[0].LineID(); !ok || got != lineSnapshot {
		t.Fatalf("expected line id clone, got %q", got)
	}
	lineID = testLiteralMutated
	if got, _ := breedingUnits[0].LineID(); got != lineSnapshot {
		t.Fatalf("expected line id clone to remain stable, got %q", got)
	}
	strainID = testLiteralMutated
	if got, _ := breedingUnits[0].StrainID(); got != strainSnapshot {
		t.Fatalf("expected strain id clone to remain stable, got %q", got)
	}
	if got, ok := breedingUnits[0].TargetLineID(); !ok || got != targetLineSnapshot {
		t.Fatalf("expected target line id clone, got %q", got)
	}
	targetLineID = testLiteralMutated
	if got, _ := breedingUnits[0].TargetLineID(); got != targetLineSnapshot {
		t.Fatalf("expected target line id clone to remain stable, got %q", got)
	}
	targetStrainID = testLiteralMutated
	if got, _ := breedingUnits[0].TargetStrainID(); got != targetStrainSnapshot {
		t.Fatalf("expected target strain clone to remain stable, got %q", got)
	}
	if intent := breedingUnits[0].PairingIntent(); intent != "outcross" {
		t.Fatalf("expected pairing intent 'outcross', got %q", intent)
	}
	if notes := breedingUnits[0].PairingNotes(); notes != "Documented pairing" {
		t.Fatalf("expected pairing notes clone, got %q", notes)
	}
	attr := breedingUnits[0].PairingAttributes()
	attr["purpose"] = testLiteralMutated
	if breeding.PairingAttributes()["purpose"] != "lineage" {
		t.Fatalf("expected pairing attributes to be cloned")
	}
	procIDs := procedures[0].OrganismIDs()
	procIDs[0] = testLiteralMutated
	if procedure.OrganismIDs[0] != "o1" {
		t.Fatalf("expected procedure slice clone")
	}
	if projectID, ok := procedures[0].ProjectID(); !ok || projectID != testProjectID {
		t.Fatalf("expected project id clone on procedure")
	}
	treatments := procedures[0].TreatmentIDs()
	if len(treatments) != 1 || treatments[0] != "t1" {
		t.Fatalf("expected treatment ids on procedure facade")
	}
	treatments[0] = testLiteralMutated
	if procedure.TreatmentIDs[0] != "t1" {
		t.Fatalf("expected procedure treatment ids clone")
	}
	obs := procedures[0].ObservationIDs()
	obs[0] = testLiteralMutated
	if procedure.ObservationIDs[0] != "obs1" {
		t.Fatalf("expected procedure observation ids clone")
	}
	projectIDs := projects[0].FacilityIDs()
	projectIDs[0] = testLiteralMutated
	if project.FacilityIDs[0] != "facility" {
		t.Fatalf("expected project facility ids clone")
	}
	protocolIDs := projects[0].ProtocolIDs()
	if len(protocolIDs) != 1 || protocolIDs[0] != "protocol" {
		t.Fatalf("expected project protocol ids clone")
	}
	protocolIDs[0] = testLiteralMutated
	if project.ProtocolIDs[0] != "protocol" {
		t.Fatalf("expected original project protocol ids to remain unchanged")
	}
	orgIDs := projects[0].OrganismIDs()
	orgIDClone := orgIDs[0]
	if len(orgIDs) != 1 || orgIDs[0] != orgIDClone {
		t.Fatalf("expected project organism ids clone")
	}
	orgIDs[0] = testLiteralMutated
	if project.OrganismIDs[0] != orgIDClone {
		t.Fatalf("expected original project organism ids to remain unchanged")
	}
	procedureIDs := projects[0].ProcedureIDs()
	if len(procedureIDs) != 1 || procedureIDs[0] != "procedure" {
		t.Fatalf("expected project procedure ids clone")
	}
	procedureIDs[0] = testLiteralMutated
	if project.ProcedureIDs[0] != "procedure" {
		t.Fatalf("expected original project procedure ids to remain unchanged")
	}
	supplyIDs := projects[0].SupplyItemIDs()
	if len(supplyIDs) != 1 || supplyIDs[0] != "supply" {
		t.Fatalf("expected project supply item ids clone")
	}
	supplyIDs[0] = testLiteralMutated
	if project.SupplyItemIDs[0] != "supply" {
		t.Fatalf("expected original project supply item ids to remain unchanged")
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
