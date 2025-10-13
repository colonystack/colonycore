package core_test

import (
	"colonycore/internal/core"
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
	"colonycore/plugins/frog"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	memory "colonycore/internal/infra/persistence/memory"
)

func TestHousingCapacityRuleBlocksOverCapacity(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	ctx := context.Background()

	housing, res, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{Name: "Tank A", FacilityID: "Greenhouse", Capacity: 1})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations: %+v", res.Violations)
	}

	frogA, res, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog A", Species: "Lithobates", Stage: domain.StageJuvenile})
	if err != nil {
		t.Fatalf("create organism A: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations for organism A: %+v", res.Violations)
	}

	frogB, res, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog B", Species: "Lithobates", Stage: domain.StageJuvenile})
	if err != nil {
		t.Fatalf("create organism B: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations for organism B: %+v", res.Violations)
	}

	if _, res, err = svc.AssignOrganismHousing(ctx, frogA.ID, housing.ID); err != nil {
		t.Fatalf("assign housing for frog A: %v", err)
	} else if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations on first assignment: %+v", res.Violations)
	}

	_, res, err = svc.AssignOrganismHousing(ctx, frogB.ID, housing.ID)
	if err == nil {
		t.Fatalf("expected error when exceeding housing capacity")
	}
	var violationErr domain.RuleViolationError
	if !AsRuleViolation(err, &violationErr) {
		t.Fatalf("expected rule violation error, got %T", err)
	}
	if !violationErr.Result.HasBlocking() {
		t.Fatalf("expected blocking violation")
	}
	if len(violationErr.Result.Violations) != 1 || violationErr.Result.Violations[0].Rule != "housing_capacity" {
		t.Fatalf("unexpected violations: %+v", violationErr.Result.Violations)
	}
}

func TestProtocolSubjectCapBlocksOverage(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	ctx := context.Background()

	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-1", Title: "Regeneration"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.ID == "" {
		t.Fatalf("expected project ID to be set")
	}

	protocol, res, err := svc.CreateProtocol(ctx, domain.Protocol{Code: "PROTO-1", Title: "Tadpole Study", MaxSubjects: 1})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations on protocol create: %+v", res.Violations)
	}

	frogA, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog A", Species: "Lithobates", ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("create organism A: %v", err)
	}
	frogB, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Frog B", Species: "Lithobates", ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("create organism B: %v", err)
	}

	if _, res, err = svc.AssignOrganismProtocol(ctx, frogA.ID, protocol.ID); err != nil {
		t.Fatalf("assign protocol to frog A: %v", err)
	} else if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations on first assignment: %+v", res.Violations)
	}

	_, res, err = svc.AssignOrganismProtocol(ctx, frogB.ID, protocol.ID)
	if err == nil {
		t.Fatalf("expected error when exceeding protocol subjects")
	}
	var violationErr domain.RuleViolationError
	if !AsRuleViolation(err, &violationErr) {
		t.Fatalf("expected rule violation error, got %T", err)
	}
	if len(violationErr.Result.Violations) == 0 || violationErr.Result.Violations[0].Rule != "protocol_subject_cap" {
		t.Fatalf("unexpected violations: %+v", violationErr.Result.Violations)
	}
}

func TestFrogPluginRegistersSchemasAndRules(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	meta, err := svc.InstallPlugin(frog.New())
	if err != nil {
		t.Fatalf("install frog plugin: %v", err)
	}
	if meta.Name != "frog" {
		t.Fatalf("unexpected plugin name: %s", meta.Name)
	}
	if _, ok := meta.Schemas["organism"]; !ok {
		t.Fatalf("expected frog plugin to register organism schema")
	}
	if len(meta.Datasets) != 1 {
		t.Fatalf("expected frog plugin to register dataset template")
	}
	templates := svc.DatasetTemplates()
	if len(templates) != 1 {
		t.Fatalf("expected dataset templates from service")
	}
	if templates[0].Slug != meta.Datasets[0].Slug {
		t.Fatalf("expected descriptors to align")
	}
	if _, ok := svc.ResolveDatasetTemplate(meta.Datasets[0].Slug); !ok {
		t.Fatalf("expected dataset template to resolve")
	}

	ctx := context.Background()
	housing, _, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{Name: "Dry Terrarium", FacilityID: "Lab", Capacity: 2, Environment: "arid"})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}

	frogA, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "DryFrog", Species: "Poison Frog"})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}

	_, res, err := svc.AssignOrganismHousing(ctx, frogA.ID, housing.ID)
	if err != nil {
		t.Fatalf("assign frog housing: %v", err)
	}
	if len(res.Violations) != 1 {
		t.Fatalf("expected single warning violation, got %+v", res.Violations)
	}
	violation := res.Violations[0]
	if violation.Severity != domain.SeverityWarn || violation.Rule != "frog_habitat_warning" {
		t.Fatalf("unexpected violation: %+v", violation)
	}
	if len(svc.RegisteredPlugins()) != 1 {
		t.Fatalf("expected one registered plugin")
	}
}

func TestServiceExtendedCRUD(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	ctx := context.Background()

	project, _, err := svc.CreateProject(ctx, domain.Project{Code: "PRJ-EXT", Title: "Extended"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	protocol, _, err := svc.CreateProtocol(ctx, domain.Protocol{Code: "PROT-EXT", Title: "Extended Protocol", MaxSubjects: 10})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	housing, _, err := svc.CreateHousingUnit(ctx, domain.HousingUnit{Name: "Humid", FacilityID: "Lab", Capacity: 4, Environment: "humid"})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}

	projID := project.ID
	protID := protocol.ID
	housingID := housing.ID

	cohort, _, err := svc.CreateCohort(ctx, domain.Cohort{Name: "Cohort", Purpose: "Study", ProjectID: &projID, HousingID: &housingID, ProtocolID: &protID})
	if err != nil {
		t.Fatalf("create cohort: %v", err)
	}

	cohortID := cohort.ID
	organismA, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "SpecimenA", Species: "Lithobates", Stage: domain.StageJuvenile, CohortID: &cohortID})
	if err != nil {
		t.Fatalf("create organismA: %v", err)
	}
	organismB, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "SpecimenB", Species: "Lithobates", Stage: domain.StageAdult, CohortID: &cohortID})
	if err != nil {
		t.Fatalf("create organismB: %v", err)
	}

	updated, res, err := svc.UpdateOrganism(ctx, organismA.ID, func(o *domain.Organism) error {
		o.Line = "LineA"
		return nil
	})
	if err != nil {
		t.Fatalf("update organism: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations on organism update: %+v", res.Violations)
	}
	if updated.Line != "LineA" {
		t.Fatalf("expected line to update, got %s", updated.Line)
	}

	breeding, _, err := svc.CreateBreedingUnit(ctx, domain.BreedingUnit{
		Name:       "Pair",
		Strategy:   "pair",
		HousingID:  &housingID,
		ProtocolID: &protID,
		FemaleIDs:  []string{organismA.ID},
		MaleIDs:    []string{organismB.ID},
	})
	if err != nil {
		t.Fatalf("create breeding unit: %v", err)
	}
	if breeding.Name == "" {
		t.Fatalf("expected breeding unit to have name")
	}

	procedure, _, err := svc.CreateProcedure(ctx, domain.Procedure{
		Name:        "Procedure",
		Status:      "scheduled",
		ScheduledAt: time.Now().Add(time.Minute),
		ProtocolID:  protID,
		OrganismIDs: []string{organismA.ID},
	})
	if err != nil {
		t.Fatalf("create procedure: %v", err)
	}

	if _, res, err := svc.UpdateProcedure(ctx, procedure.ID, func(p *domain.Procedure) error {
		p.Status = "completed"
		return nil
	}); err != nil {
		t.Fatalf("update procedure: %v", err)
	} else if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations on procedure update: %+v", res.Violations)
	}

	if _, err := svc.DeleteProcedure(ctx, procedure.ID); err != nil {
		t.Fatalf("delete procedure: %v", err)
	}
	if _, err := svc.DeleteOrganism(ctx, organismB.ID); err != nil {
		t.Fatalf("delete organism: %v", err)
	}
}

func TestServiceConstructorAndStore(t *testing.T) {
	store := core.NewMemoryStore(core.NewRulesEngine())
	svc := core.NewService(store)
	if svc.Store() != store {
		t.Fatalf("expected Store to return provided memory store")
	}
}

// AsRuleViolation unwraps errors into a RuleViolationError when possible.
func AsRuleViolation(err error, target *domain.RuleViolationError) bool {
	if err == nil {
		return false
	}
	var rv domain.RuleViolationError
	if errors.As(err, &rv) {
		*target = rv
		return true
	}
	return false
}

func TestInstallPluginValidations(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	var nilPlugin pluginapi.Plugin
	if _, err := svc.InstallPlugin(nilPlugin); err == nil {
		t.Fatalf("expected error when plugin is nil")
	}
	if _, err := svc.InstallPlugin(frog.New()); err != nil {
		t.Fatalf("install frog plugin: %v", err)
	}
	if _, err := svc.InstallPlugin(frog.New()); err == nil {
		t.Fatalf("expected duplicate plugin error")
	}
}

func TestServiceAssignInvalidReferences(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	ctx := context.Background()
	organism, _, err := svc.CreateOrganism(ctx, domain.Organism{Name: "Lonely", Species: "Frog"})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if _, _, err := svc.AssignOrganismHousing(ctx, organism.ID, "missing"); err == nil {
		t.Fatalf("expected housing assignment error")
	} else if !strings.Contains(err.Error(), string(domain.EntityHousingUnit)) {
		t.Fatalf("unexpected housing error: %v", err)
	}
	if _, _, err := svc.AssignOrganismProtocol(ctx, organism.ID, "missing"); err == nil {
		t.Fatalf("expected protocol assignment error")
	} else if !strings.Contains(err.Error(), string(domain.EntityProtocol)) {
		t.Fatalf("unexpected protocol error: %v", err)
	}
}

func TestServiceClockAndLoggerOptions(t *testing.T) {
	formatProvider := datasetapi.GetFormatProvider()
	engine := core.NewRulesEngine()
	store := clocklessStore{inner: core.NewMemoryStore(engine)}
	freeze := time.Date(2024, 4, 20, 12, 34, 56, 0, time.UTC)
	logger := &recordingLogger{}
	svc := core.NewService(store, core.WithClock(core.ClockFunc(func() time.Time { return freeze })), core.WithLogger(logger))

	if _, err := svc.InstallPlugin(testClockPlugin{}); err != nil {
		t.Fatalf("install test plugin: %v", err)
	}
	templates := svc.DatasetTemplates()
	if len(templates) != 1 {
		t.Fatalf("expected one dataset template, got %d", len(templates))
	}
	template, ok := svc.ResolveDatasetTemplate(templates[0].Slug)
	if !ok {
		t.Fatalf("resolve dataset template: %v", templates[0])
	}
	result, errs, err := template.Run(context.Background(), nil, datasetapi.Scope{}, formatProvider.JSON())
	if err != nil {
		t.Fatalf("run dataset: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %+v", errs)
	}
	if !result.GeneratedAt.Equal(freeze) {
		t.Fatalf("expected generated at %v, got %v", freeze, result.GeneratedAt)
	}
	if !logger.infoCalled {
		t.Fatalf("expected logger info to be called")
	}
}

type recordingLogger struct {
	infoCalled bool
}

func (l *recordingLogger) Debug(string, ...any) {}
func (l *recordingLogger) Info(string, ...any)  { l.infoCalled = true }
func (l *recordingLogger) Warn(string, ...any)  {}
func (l *recordingLogger) Error(string, ...any) {}

type clocklessStore struct {
	inner *memory.Store
}

func (s clocklessStore) RunInTransaction(ctx context.Context, fn func(domain.Transaction) error) (domain.Result, error) {
	return s.inner.RunInTransaction(ctx, fn)
}

func (s clocklessStore) View(ctx context.Context, fn func(domain.TransactionView) error) error {
	return s.inner.View(ctx, fn)
}

func (s clocklessStore) GetOrganism(id string) (domain.Organism, bool) {
	return s.inner.GetOrganism(id)
}

func (s clocklessStore) ListOrganisms() []domain.Organism {
	return s.inner.ListOrganisms()
}

func (s clocklessStore) GetHousingUnit(id string) (domain.HousingUnit, bool) {
	return s.inner.GetHousingUnit(id)
}

func (s clocklessStore) ListHousingUnits() []domain.HousingUnit {
	return s.inner.ListHousingUnits()
}

func (s clocklessStore) ListCohorts() []domain.Cohort {
	return s.inner.ListCohorts()
}

func (s clocklessStore) ListProtocols() []domain.Protocol {
	return s.inner.ListProtocols()
}

func (s clocklessStore) ListProjects() []domain.Project {
	return s.inner.ListProjects()
}

func (s clocklessStore) ListBreedingUnits() []domain.BreedingUnit {
	return s.inner.ListBreedingUnits()
}

func (s clocklessStore) ListProcedures() []domain.Procedure {
	return s.inner.ListProcedures()
}

func (s clocklessStore) RulesEngine() *domain.RulesEngine {
	return s.inner.RulesEngine()
}

type testClockPlugin struct{}

func (testClockPlugin) Name() string    { return "clock" }
func (testClockPlugin) Version() string { return "v1" }

func (testClockPlugin) Register(registry pluginapi.Registry) error {
	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()
	return registry.RegisterDatasetTemplate(datasetapi.Template{
		Key:           "now",
		Version:       "v1",
		Title:         "Now",
		Description:   "returns the current time",
		Dialect:       dialectProvider.SQL(),
		Query:         "SELECT now()",
		OutputFormats: []datasetapi.Format{formatProvider.JSON()},
		Columns:       []datasetapi.Column{{Name: "now", Type: "timestamp"}},
		Binder: func(env datasetapi.Environment) (datasetapi.Runner, error) {
			return func(_ context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
				now := env.Now
				if now == nil {
					now = func() time.Time { return time.Now().UTC() }
				}
				timestamp := now().UTC()
				return datasetapi.RunResult{
					Schema:      req.Template.Columns,
					Rows:        []datasetapi.Row{{"now": timestamp}},
					GeneratedAt: timestamp,
					Format:      formatProvider.JSON(),
				}, nil
			}, nil
		},
	})
}
