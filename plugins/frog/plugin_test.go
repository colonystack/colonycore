package frog

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
	"colonycore/plugins/testhelper"
)

type stubRegistry struct {
	schemas  map[string]map[string]any
	rules    []pluginapi.Rule
	datasets []datasetapi.Template
}

func newStubRegistry() *stubRegistry {
	return &stubRegistry{schemas: make(map[string]map[string]any)}
}

func (r *stubRegistry) RegisterSchema(entity string, schema map[string]any) {
	r.schemas[entity] = schema
}

func (r *stubRegistry) RegisterRule(rule pluginapi.Rule) {
	r.rules = append(r.rules, rule)
}

func (r *stubRegistry) RegisterDatasetTemplate(template datasetapi.Template) error {
	r.datasets = append(r.datasets, template)
	return nil
}

type stubView struct {
	organisms []datasetapi.Organism
	housing   map[string]datasetapi.HousingUnit
}

func newStubView() *stubView {
	return &stubView{housing: make(map[string]datasetapi.HousingUnit)}
}

func (s *stubView) ListOrganisms() []datasetapi.Organism {
	return append([]datasetapi.Organism(nil), s.organisms...)
}

func (s *stubView) ListHousingUnits() []datasetapi.HousingUnit {
	units := make([]datasetapi.HousingUnit, 0, len(s.housing))
	for _, unit := range s.housing {
		units = append(units, unit)
	}
	return units
}

func (s *stubView) ListProtocols() []datasetapi.Protocol       { return nil }
func (s *stubView) ListFacilities() []datasetapi.Facility      { return nil }
func (s *stubView) ListTreatments() []datasetapi.Treatment     { return nil }
func (s *stubView) ListObservations() []datasetapi.Observation { return nil }
func (s *stubView) ListSamples() []datasetapi.Sample           { return nil }
func (s *stubView) ListPermits() []datasetapi.Permit           { return nil }
func (s *stubView) ListProjects() []datasetapi.Project         { return nil }
func (s *stubView) ListSupplyItems() []datasetapi.SupplyItem   { return nil }

func (s *stubView) FindOrganism(id string) (datasetapi.Organism, bool) {
	for _, org := range s.organisms {
		if org.ID() == id {
			return org, true
		}
	}
	return nil, false
}

func (s *stubView) FindHousingUnit(id string) (datasetapi.HousingUnit, bool) {
	unit, ok := s.housing[id]
	return unit, ok
}

func (s *stubView) FindFacility(string) (datasetapi.Facility, bool)       { return nil, false }
func (s *stubView) FindTreatment(string) (datasetapi.Treatment, bool)     { return nil, false }
func (s *stubView) FindObservation(string) (datasetapi.Observation, bool) { return nil, false }
func (s *stubView) FindSample(string) (datasetapi.Sample, bool)           { return nil, false }
func (s *stubView) FindPermit(string) (datasetapi.Permit, bool)           { return nil, false }
func (s *stubView) FindSupplyItem(string) (datasetapi.SupplyItem, bool)   { return nil, false }

type stubStore struct {
	view datasetapi.TransactionView
}

func (s stubStore) View(_ context.Context, fn func(datasetapi.TransactionView) error) error {
	return fn(s.view)
}

func (stubStore) GetOrganism(string) (datasetapi.Organism, bool) { return nil, false }
func (stubStore) ListOrganisms() []datasetapi.Organism           { return nil }
func (stubStore) GetHousingUnit(string) (datasetapi.HousingUnit, bool) {
	return nil, false
}
func (stubStore) ListHousingUnits() []datasetapi.HousingUnit     { return nil }
func (stubStore) GetFacility(string) (datasetapi.Facility, bool) { return nil, false }
func (stubStore) ListFacilities() []datasetapi.Facility          { return nil }
func (stubStore) ListCohorts() []datasetapi.Cohort               { return nil }
func (stubStore) ListTreatments() []datasetapi.Treatment         { return nil }
func (stubStore) ListObservations() []datasetapi.Observation     { return nil }
func (stubStore) ListSamples() []datasetapi.Sample               { return nil }
func (stubStore) ListProtocols() []datasetapi.Protocol           { return nil }
func (stubStore) GetPermit(string) (datasetapi.Permit, bool)     { return nil, false }
func (stubStore) ListPermits() []datasetapi.Permit               { return nil }
func (stubStore) ListProjects() []datasetapi.Project             { return nil }
func (stubStore) ListBreedingUnits() []datasetapi.BreedingUnit   { return nil }
func (stubStore) ListProcedures() []datasetapi.Procedure         { return nil }
func (stubStore) ListSupplyItems() []datasetapi.SupplyItem       { return nil }

func TestPluginRegistration(t *testing.T) {
	plugin := New()
	registry := newStubRegistry()

	if err := plugin.Register(registry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	if _, ok := registry.schemas["organism"]; !ok {
		t.Fatalf("expected organism schema to be registered")
	}
	if len(registry.rules) != 1 {
		t.Fatalf("expected single rule registration, got %d", len(registry.rules))
	}
	if len(registry.datasets) != 1 {
		t.Fatalf("expected single dataset registration, got %d", len(registry.datasets))
	}
}

func TestFrogHabitatRuleOutcomes(t *testing.T) {
	rule := frogHabitatRule{}
	aquaticID := "AQUA"
	dryID := "DRY"
	view := fakeView{
		organisms: []pluginapi.OrganismView{
			stubOrganism{id: "frog-safe", species: "Tree Frog", housingID: &aquaticID},
			stubOrganism{id: "frog-risk", species: "Poison Frog", housingID: &dryID},
			stubOrganism{id: "not-frog", species: "Gecko", housingID: &dryID},
		},
		housing: []pluginapi.HousingUnitView{
			stubHousing{id: aquaticID, environment: "humid"},
			stubHousing{id: dryID, environment: "dry"},
		},
	}

	res, err := rule.Evaluate(context.Background(), view, nil)
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}

	if len(res.Violations()) != 1 {
		t.Fatalf("expected exactly one violation, got %d", len(res.Violations()))
	}
	violation := res.Violations()[0]
	if violation.EntityID() != "frog-risk" {
		t.Fatalf("expected violation for frog-risk, got %s", violation.EntityID())
	}
	severities := pluginapi.NewSeverityContext()
	expectedSeverity := severities.Warn()
	if violation.Severity() != pluginapi.Severity(expectedSeverity.String()) {
		t.Fatalf("expected warning severity, got %v", violation.Severity())
	}
}

func TestFrogPopulationBinderFilters(t *testing.T) {
	plugin := New()
	registry := newStubRegistry()
	if err := plugin.Register(registry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	template := registry.datasets[0]
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	projectA, projectB := "project-a", "project-b"
	protocol := "protocol"
	housing := "housing"
	stages := testhelper.LifecycleStages()
	retired := stages.Retired
	adult := stages.Adult
	organisms := testhelper.Organisms(
		testhelper.OrganismFixtureConfig{
			BaseFixture: testhelper.BaseFixture{ID: "alpha", UpdatedAt: now},
			Name:        "Alpha",
			Species:     "Tree Frog",
			Stage:       adult,
			ProjectID:   &projectA,
			ProtocolID:  &protocol,
			HousingID:   &housing,
		},
		testhelper.OrganismFixtureConfig{
			BaseFixture: testhelper.BaseFixture{ID: "bravo", UpdatedAt: now.Add(time.Second)},
			Name:        "Bravo",
			Species:     "Tree Frog",
			Stage:       adult,
			ProjectID:   &projectB,
			ProtocolID:  &protocol,
			HousingID:   &housing,
		},
		testhelper.OrganismFixtureConfig{
			BaseFixture: testhelper.BaseFixture{ID: "charlie", UpdatedAt: now.Add(2 * time.Second)},
			Name:        "Charlie",
			Species:     "Tree Frog",
			Stage:       retired,
			ProjectID:   &projectA,
		},
		testhelper.OrganismFixtureConfig{
			BaseFixture: testhelper.BaseFixture{ID: "other", UpdatedAt: now},
			Name:        "Gecko",
			Species:     "Gecko",
			Stage:       adult,
		},
	)

	view := newStubView()
	view.organisms = organisms
	view.housing[housing] = testhelper.HousingUnit(testhelper.HousingUnitFixtureConfig{
		BaseFixture: testhelper.BaseFixture{ID: housing},
		Name:        "Habitat",
		Environment: "humid",
	})
	store := stubStore{view: view}

	env := datasetapi.Environment{Store: store, Now: func() time.Time { return now }}
	runner, err := frogPopulationBinder(env)
	if err != nil {
		t.Fatalf("create binder: %v", err)
	}

	request := datasetapi.RunRequest{
		Template: datasetapi.TemplateDescriptor{Columns: template.Columns},
		Scope:    datasetapi.Scope{ProjectIDs: []string{projectA}},
		Parameters: map[string]any{
			"include_retired": false,
		},
	}
	result, err := runner(context.Background(), request)
	if err != nil {
		t.Fatalf("run binder: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected single row for project scope, got %d", len(result.Rows))
	}
	if result.Rows[0]["organism_id"] != "alpha" {
		t.Fatalf("unexpected organism: %+v", result.Rows[0])
	}

	request.Parameters["include_retired"] = true
	result, err = runner(context.Background(), request)
	if err != nil {
		t.Fatalf("run binder include retired: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected two rows with retired included, got %d", len(result.Rows))
	}

	request.Parameters = map[string]any{"stage": string(testhelper.LifecycleStages().Larva)}
	result, err = runner(context.Background(), request)
	if err != nil {
		t.Fatalf("run binder stage filter: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Fatalf("expected no larva results, got %d", len(result.Rows))
	}
}
