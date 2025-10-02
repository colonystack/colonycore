package frog

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
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

func (s *stubView) ListProtocols() []datasetapi.Protocol { return nil }

func (s *stubView) FindOrganism(id string) (datasetapi.Organism, bool) {
	for _, org := range s.organisms {
		if org.ID == id {
			return org, true
		}
	}
	return datasetapi.Organism{}, false
}

func (s *stubView) FindHousingUnit(id string) (datasetapi.HousingUnit, bool) {
	unit, ok := s.housing[id]
	return unit, ok
}

type stubStore struct {
	view datasetapi.TransactionView
}

func (s stubStore) RunInTransaction(context.Context, func(datasetapi.Transaction) error) (datasetapi.Result, error) {
	return datasetapi.Result{}, nil
}

func (s stubStore) View(_ context.Context, fn func(datasetapi.TransactionView) error) error {
	return fn(s.view)
}

func (stubStore) GetOrganism(string) (datasetapi.Organism, bool) { return datasetapi.Organism{}, false }
func (stubStore) ListOrganisms() []datasetapi.Organism           { return nil }
func (stubStore) GetHousingUnit(string) (datasetapi.HousingUnit, bool) {
	return datasetapi.HousingUnit{}, false
}
func (stubStore) ListHousingUnits() []datasetapi.HousingUnit   { return nil }
func (stubStore) ListCohorts() []datasetapi.Cohort             { return nil }
func (stubStore) ListProtocols() []datasetapi.Protocol         { return nil }
func (stubStore) ListProjects() []datasetapi.Project           { return nil }
func (stubStore) ListBreedingUnits() []datasetapi.BreedingUnit { return nil }
func (stubStore) ListProcedures() []datasetapi.Procedure       { return nil }

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
	view := newStubView()

	aquatic := datasetapi.HousingUnit{Base: datasetapi.Base{ID: "AQUA"}, Environment: "humid"}
	dry := datasetapi.HousingUnit{Base: datasetapi.Base{ID: "DRY"}, Environment: "dry"}
	view.housing[aquatic.ID] = aquatic
	view.housing[dry.ID] = dry

	frogSafe := datasetapi.Organism{Base: datasetapi.Base{ID: "frog-safe"}, Species: "Tree Frog", HousingID: &aquatic.ID}
	frogUnsafe := datasetapi.Organism{Base: datasetapi.Base{ID: "frog-risk"}, Species: "Poison Frog", HousingID: &dry.ID}
	notFrog := datasetapi.Organism{Base: datasetapi.Base{ID: "not-frog"}, Species: "Gecko", HousingID: &dry.ID}
	view.organisms = append(view.organisms, frogSafe, frogUnsafe, notFrog)

	res, err := rule.Evaluate(context.Background(), view, nil)
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}

	if len(res.Violations) != 1 {
		t.Fatalf("expected exactly one violation, got %d", len(res.Violations))
	}
	violation := res.Violations[0]
	if violation.EntityID != frogUnsafe.ID {
		t.Fatalf("expected violation for %s, got %s", frogUnsafe.ID, violation.EntityID)
	}
	if violation.Severity != pluginapi.SeverityWarn {
		t.Fatalf("expected warning severity, got %v", violation.Severity)
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
	retired := datasetapi.StageRetired
	adult := datasetapi.StageAdult
	organisms := []datasetapi.Organism{
		{Base: datasetapi.Base{ID: "alpha", UpdatedAt: now}, Name: "Alpha", Species: "Tree Frog", Stage: adult, ProjectID: &projectA, ProtocolID: &protocol, HousingID: &housing},
		{Base: datasetapi.Base{ID: "bravo", UpdatedAt: now.Add(time.Second)}, Name: "Bravo", Species: "Tree Frog", Stage: adult, ProjectID: &projectB, ProtocolID: &protocol, HousingID: &housing},
		{Base: datasetapi.Base{ID: "charlie", UpdatedAt: now.Add(2 * time.Second)}, Name: "Charlie", Species: "Tree Frog", Stage: retired, ProjectID: &projectA},
		{Base: datasetapi.Base{ID: "other", UpdatedAt: now}, Name: "Gecko", Species: "Gecko", Stage: adult},
	}

	view := newStubView()
	view.organisms = organisms
	view.housing[housing] = datasetapi.HousingUnit{Base: datasetapi.Base{ID: housing}, Environment: "humid"}
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

	request.Parameters = map[string]any{"stage": string(datasetapi.StageLarva)}
	result, err = runner(context.Background(), request)
	if err != nil {
		t.Fatalf("run binder stage filter: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Fatalf("expected no larva results, got %d", len(result.Rows))
	}
}
