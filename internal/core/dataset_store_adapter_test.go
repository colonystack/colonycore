package core

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

func TestDatasetPersistentStoreAdapter(t *testing.T) {
	now := time.Date(2024, 7, 1, 8, 0, 0, 0, time.UTC)
	const (
		organismID = "organism"
		housingID  = "housing"
		protocolID = "protocol"
		projectID  = "project"
		cohortID   = "cohort"
	)
	cohort := cohortID
	housingRef := housingID
	protocolRef := protocolID
	projectRef := projectID
	junkAttr := map[string]any{"flag": true}
	organism := domain.Organism{
		Base:       domain.Base{ID: organismID, CreatedAt: now.Add(-time.Hour), UpdatedAt: now},
		Name:       "Alpha",
		Species:    "Frog",
		Stage:      domain.StageAdult,
		CohortID:   &cohort,
		HousingID:  &housingRef,
		ProtocolID: &protocolRef,
		ProjectID:  &projectRef,
		Attributes: junkAttr,
	}
	unit := domain.HousingUnit{Base: domain.Base{ID: housingID}, Name: "Hab", Environment: "humid"}
	protocol := domain.Protocol{Base: domain.Base{ID: protocolID}, Code: "P", Title: "Protocol", Description: "Desc", MaxSubjects: 10}
	project := domain.Project{Base: domain.Base{ID: projectID}, Code: "PR", Title: "Project", Description: "Research"}
	cohortEntity := domain.Cohort{Base: domain.Base{ID: cohortID}, Name: "Group", Purpose: "Study", ProjectID: &projectRef, HousingID: &housingRef, ProtocolID: &protocolRef}
	breeding := domain.BreedingUnit{Base: domain.Base{ID: "breeding"}, Name: "Pair", Strategy: "pair", HousingID: &housingRef, ProtocolID: &protocolRef, FemaleIDs: []string{"f"}, MaleIDs: []string{"m"}}
	procedure := domain.Procedure{Base: domain.Base{ID: "procedure"}, Name: "Proc", Status: "scheduled", ScheduledAt: now.Add(time.Hour), ProtocolID: protocolID, CohortID: &cohort, OrganismIDs: []string{organismID}}

	fake := &fakePersistentStore{
		organisms:     []domain.Organism{organism},
		housingUnits:  []domain.HousingUnit{unit},
		protocols:     []domain.Protocol{protocol},
		projects:      []domain.Project{project},
		cohorts:       []domain.Cohort{cohortEntity},
		breedingUnits: []domain.BreedingUnit{breeding},
		procedures:    []domain.Procedure{procedure},
	}
	adapter := newDatasetPersistentStore(fake)
	if adapter == nil {
		t.Fatalf("expected adapter instance")
	}

	adaptedOrg, ok := adapter.GetOrganism(organismID)
	expectedStage := datasetapi.LifecycleStage(datasetapi.NewLifecycleStageContext().Adult().String())
	if !ok || adaptedOrg.ID() != organismID || adaptedOrg.Stage() != expectedStage {
		t.Fatalf("expected converted organism")
	}
	attrs := adaptedOrg.Attributes()
	attrs["flag"] = false
	if fake.organisms[0].Attributes["flag"].(bool) != true {
		t.Fatalf("expected original organism attributes untouched")
	}

	organisms := adapter.ListOrganisms()
	if len(organisms) != 1 || organisms[0].ID() != organismID {
		t.Fatalf("expected converted organism slice")
	}

	if housing, ok := adapter.GetHousingUnit(housingID); !ok || housing.ID() != housingID {
		t.Fatalf("expected converted housing unit")
	}
	housingUnits := adapter.ListHousingUnits()
	if len(housingUnits) != 1 || housingUnits[0].Environment() != unit.Environment {
		t.Fatalf("expected converted housing slice")
	}

	cohorts := adapter.ListCohorts()
	if len(cohorts) != 1 {
		t.Fatalf("expected cohort conversion")
	}
	projectIdent, ok := cohorts[0].ProjectID()
	if !ok || projectIdent != projectID {
		t.Fatalf("expected cohort project id clone")
	}
	projectRef = testLiteralMutated
	if retained, _ := cohorts[0].ProjectID(); retained != projectIdent {
		t.Fatalf("expected cohort project id to be immutable")
	}
	projectRef = projectID

	if protocols := adapter.ListProtocols(); len(protocols) != 1 || protocols[0].MaxSubjects() != protocol.MaxSubjects {
		t.Fatalf("expected protocol conversion")
	}
	if projects := adapter.ListProjects(); len(projects) != 1 || projects[0].Title() != project.Title {
		t.Fatalf("expected project conversion")
	}
	breedingUnits := adapter.ListBreedingUnits()
	if len(breedingUnits) != 1 || breedingUnits[0].FemaleIDs()[0] != "f" {
		t.Fatalf("expected breeding conversion")
	}
	females := breedingUnits[0].FemaleIDs()
	females[0] = testLiteralMutated
	if fake.breedingUnits[0].FemaleIDs[0] != "f" {
		t.Fatalf("expected breeding slice clone")
	}
	procedures := adapter.ListProcedures()
	if len(procedures) != 1 || procedures[0].OrganismIDs()[0] != organismID {
		t.Fatalf("expected procedure conversion")
	}
	ids := procedures[0].OrganismIDs()
	ids[0] = testLiteralMutated
	if fake.procedures[0].OrganismIDs[0] != organismID {
		t.Fatalf("expected procedure slice clone")
	}

	if err := adapter.View(context.Background(), func(view datasetapi.TransactionView) error {
		orgs := view.ListOrganisms()
		if len(orgs) != 1 || orgs[0].ID() != organismID {
			t.Fatalf("expected view organisms conversion")
		}
		if _, ok := view.FindOrganism("missing"); ok {
			t.Fatalf("expected missing organism lookup to fail")
		}
		housingSlice := view.ListHousingUnits()
		if len(housingSlice) != 1 || housingSlice[0].ID() != housingID {
			t.Fatalf("expected view housing conversion")
		}
		if _, ok := view.FindHousingUnit("missing"); ok {
			t.Fatalf("expected missing housing lookup to fail")
		}
		if len(view.ListProtocols()) != 1 {
			t.Fatalf("expected view protocols conversion")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
	if !fake.viewCalled {
		t.Fatalf("expected underlying store view to be invoked")
	}

	if err := adapter.View(context.Background(), nil); err != nil {
		t.Fatalf("expected nil fn to succeed")
	}
}

type fakePersistentStore struct {
	organisms     []domain.Organism
	housingUnits  []domain.HousingUnit
	facilities    []domain.Facility
	protocols     []domain.Protocol
	projects      []domain.Project
	cohorts       []domain.Cohort
	breedingUnits []domain.BreedingUnit
	procedures    []domain.Procedure
	treatments    []domain.Treatment
	observations  []domain.Observation
	samples       []domain.Sample
	permits       []domain.Permit
	supplyItems   []domain.SupplyItem
	viewCalled    bool
}

func (f *fakePersistentStore) RunInTransaction(context.Context, func(domain.Transaction) error) (domain.Result, error) {
	return domain.Result{}, nil
}

func (f *fakePersistentStore) View(_ context.Context, fn func(domain.TransactionView) error) error {
	f.viewCalled = true
	if fn == nil {
		return nil
	}
	return fn(fakeTransactionView{store: f})
}

func (f *fakePersistentStore) GetOrganism(id string) (domain.Organism, bool) {
	for _, org := range f.organisms {
		if org.ID == id {
			return org, true
		}
	}
	return domain.Organism{}, false
}

func (f *fakePersistentStore) ListOrganisms() []domain.Organism {
	return append([]domain.Organism(nil), f.organisms...)
}

func (f *fakePersistentStore) GetHousingUnit(id string) (domain.HousingUnit, bool) {
	for _, unit := range f.housingUnits {
		if unit.ID == id {
			return unit, true
		}
	}
	return domain.HousingUnit{}, false
}

func (f *fakePersistentStore) ListHousingUnits() []domain.HousingUnit {
	return append([]domain.HousingUnit(nil), f.housingUnits...)
}

func (f *fakePersistentStore) GetFacility(id string) (domain.Facility, bool) {
	for _, fac := range f.facilities {
		if fac.ID == id {
			return fac, true
		}
	}
	return domain.Facility{}, false
}

func (f *fakePersistentStore) ListFacilities() []domain.Facility {
	return append([]domain.Facility(nil), f.facilities...)
}

func (f *fakePersistentStore) ListCohorts() []domain.Cohort {
	return append([]domain.Cohort(nil), f.cohorts...)
}

func (f *fakePersistentStore) ListProtocols() []domain.Protocol {
	return append([]domain.Protocol(nil), f.protocols...)
}

func (f *fakePersistentStore) ListTreatments() []domain.Treatment {
	return append([]domain.Treatment(nil), f.treatments...)
}

func (f *fakePersistentStore) ListObservations() []domain.Observation {
	return append([]domain.Observation(nil), f.observations...)
}

func (f *fakePersistentStore) ListSamples() []domain.Sample {
	return append([]domain.Sample(nil), f.samples...)
}

func (f *fakePersistentStore) GetPermit(id string) (domain.Permit, bool) {
	for _, permit := range f.permits {
		if permit.ID == id {
			return permit, true
		}
	}
	return domain.Permit{}, false
}

func (f *fakePersistentStore) ListPermits() []domain.Permit {
	return append([]domain.Permit(nil), f.permits...)
}

func (f *fakePersistentStore) ListProjects() []domain.Project {
	return append([]domain.Project(nil), f.projects...)
}

func (f *fakePersistentStore) ListBreedingUnits() []domain.BreedingUnit {
	return append([]domain.BreedingUnit(nil), f.breedingUnits...)
}

func (f *fakePersistentStore) ListProcedures() []domain.Procedure {
	return append([]domain.Procedure(nil), f.procedures...)
}

func (f *fakePersistentStore) ListSupplyItems() []domain.SupplyItem {
	return append([]domain.SupplyItem(nil), f.supplyItems...)
}

type fakeTransactionView struct {
	store *fakePersistentStore
}

func (v fakeTransactionView) ListOrganisms() []domain.Organism { return v.store.ListOrganisms() }
func (v fakeTransactionView) ListHousingUnits() []domain.HousingUnit {
	return v.store.ListHousingUnits()
}
func (v fakeTransactionView) ListFacilities() []domain.Facility {
	return v.store.ListFacilities()
}
func (v fakeTransactionView) ListProtocols() []domain.Protocol { return v.store.ListProtocols() }
func (v fakeTransactionView) ListTreatments() []domain.Treatment {
	return v.store.ListTreatments()
}
func (v fakeTransactionView) ListObservations() []domain.Observation {
	return v.store.ListObservations()
}
func (v fakeTransactionView) ListSamples() []domain.Sample   { return v.store.ListSamples() }
func (v fakeTransactionView) ListPermits() []domain.Permit   { return v.store.ListPermits() }
func (v fakeTransactionView) ListProjects() []domain.Project { return v.store.ListProjects() }
func (v fakeTransactionView) ListSupplyItems() []domain.SupplyItem {
	return v.store.ListSupplyItems()
}

func (v fakeTransactionView) FindOrganism(id string) (domain.Organism, bool) {
	return v.store.GetOrganism(id)
}

func (v fakeTransactionView) FindHousingUnit(id string) (domain.HousingUnit, bool) {
	return v.store.GetHousingUnit(id)
}

func (v fakeTransactionView) FindFacility(id string) (domain.Facility, bool) {
	return v.store.GetFacility(id)
}

func (v fakeTransactionView) FindTreatment(id string) (domain.Treatment, bool) {
	for _, t := range v.store.treatments {
		if t.ID == id {
			return t, true
		}
	}
	return domain.Treatment{}, false
}

func (v fakeTransactionView) FindObservation(id string) (domain.Observation, bool) {
	for _, o := range v.store.observations {
		if o.ID == id {
			return o, true
		}
	}
	return domain.Observation{}, false
}

func (v fakeTransactionView) FindSample(id string) (domain.Sample, bool) {
	for _, s := range v.store.samples {
		if s.ID == id {
			return s, true
		}
	}
	return domain.Sample{}, false
}

func (v fakeTransactionView) FindPermit(id string) (domain.Permit, bool) {
	return v.store.GetPermit(id)
}

func (v fakeTransactionView) FindSupplyItem(id string) (domain.SupplyItem, bool) {
	for _, s := range v.store.supplyItems {
		if s.ID == id {
			return s, true
		}
	}
	return domain.SupplyItem{}, false
}
