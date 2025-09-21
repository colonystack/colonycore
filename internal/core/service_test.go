package core_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"colonycore/internal/core"
	"colonycore/internal/plugins/frog"
)

func TestHousingCapacityRuleBlocksOverCapacity(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	ctx := context.Background()

	housing, res, err := svc.CreateHousingUnit(ctx, core.HousingUnit{Name: "Tank A", Facility: "Greenhouse", Capacity: 1})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations: %+v", res.Violations)
	}

	frogA, res, err := svc.CreateOrganism(ctx, core.Organism{Name: "Frog A", Species: "Lithobates", Stage: core.StageJuvenile})
	if err != nil {
		t.Fatalf("create organism A: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations for organism A: %+v", res.Violations)
	}

	frogB, res, err := svc.CreateOrganism(ctx, core.Organism{Name: "Frog B", Species: "Lithobates", Stage: core.StageJuvenile})
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
	var violationErr core.RuleViolationError
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

	project, _, err := svc.CreateProject(ctx, core.Project{Code: "PRJ-1", Title: "Regeneration"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.ID == "" {
		t.Fatalf("expected project ID to be set")
	}

	protocol, res, err := svc.CreateProtocol(ctx, core.Protocol{Code: "PROTO-1", Title: "Tadpole Study", MaxSubjects: 1})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations on protocol create: %+v", res.Violations)
	}

	frogA, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Frog A", Species: "Lithobates", ProjectID: &project.ID})
	if err != nil {
		t.Fatalf("create organism A: %v", err)
	}
	frogB, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Frog B", Species: "Lithobates", ProjectID: &project.ID})
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
	var violationErr core.RuleViolationError
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

	ctx := context.Background()
	housing, _, err := svc.CreateHousingUnit(ctx, core.HousingUnit{Name: "Dry Terrarium", Facility: "Lab", Capacity: 2, Environment: "arid"})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}

	frogA, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "DryFrog", Species: "Poison Frog"})
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
	if violation.Severity != core.SeverityWarn || violation.Rule != "frog_habitat_warning" {
		t.Fatalf("unexpected violation: %+v", violation)
	}
	if len(svc.RegisteredPlugins()) != 1 {
		t.Fatalf("expected one registered plugin")
	}
}
func TestServiceExtendedCRUD(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	ctx := context.Background()

	project, _, err := svc.CreateProject(ctx, core.Project{Code: "PRJ-EXT", Title: "Extended"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	protocol, _, err := svc.CreateProtocol(ctx, core.Protocol{Code: "PROT-EXT", Title: "Extended Protocol", MaxSubjects: 10})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	housing, _, err := svc.CreateHousingUnit(ctx, core.HousingUnit{Name: "Humid", Facility: "Lab", Capacity: 4, Environment: "humid"})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}

	projID := project.ID
	protID := protocol.ID
	housingID := housing.ID

	cohort, _, err := svc.CreateCohort(ctx, core.Cohort{Name: "Cohort", Purpose: "Study", ProjectID: &projID, HousingID: &housingID, ProtocolID: &protID})
	if err != nil {
		t.Fatalf("create cohort: %v", err)
	}

	cohortID := cohort.ID
	organismA, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "SpecimenA", Species: "Lithobates", Stage: core.StageJuvenile, CohortID: &cohortID})
	if err != nil {
		t.Fatalf("create organismA: %v", err)
	}
	organismB, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "SpecimenB", Species: "Lithobates", Stage: core.StageAdult, CohortID: &cohortID})
	if err != nil {
		t.Fatalf("create organismB: %v", err)
	}

	updated, res, err := svc.UpdateOrganism(ctx, organismA.ID, func(o *core.Organism) error {
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

	breeding, _, err := svc.CreateBreedingUnit(ctx, core.BreedingUnit{
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

	procedure, _, err := svc.CreateProcedure(ctx, core.Procedure{
		Name:        "Procedure",
		Status:      "scheduled",
		ScheduledAt: time.Now().Add(time.Minute),
		ProtocolID:  protID,
		OrganismIDs: []string{organismA.ID},
	})
	if err != nil {
		t.Fatalf("create procedure: %v", err)
	}

	if _, res, err := svc.UpdateProcedure(ctx, procedure.ID, func(p *core.Procedure) error {
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
func AsRuleViolation(err error, target *core.RuleViolationError) bool {
	if err == nil {
		return false
	}
	if rv, ok := err.(core.RuleViolationError); ok {
		*target = rv
		return true
	}
	return false
}

func TestInstallPluginValidations(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	var nilPlugin core.Plugin
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
	organism, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Lonely", Species: "Frog"})
	if err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if _, _, err := svc.AssignOrganismHousing(ctx, organism.ID, "missing"); err == nil {
		t.Fatalf("expected housing assignment error")
	} else if !strings.Contains(err.Error(), string(core.EntityHousingUnit)) {
		t.Fatalf("unexpected housing error: %v", err)
	}
	if _, _, err := svc.AssignOrganismProtocol(ctx, organism.ID, "missing"); err == nil {
		t.Fatalf("expected protocol assignment error")
	} else if !strings.Contains(err.Error(), string(core.EntityProtocol)) {
		t.Fatalf("unexpected protocol error: %v", err)
	}
}
