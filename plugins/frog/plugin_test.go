package frog

import (
	"context"
	"testing"
	"time"

	"colonycore/internal/core"
)

func TestPluginRegistration(t *testing.T) {
	plugin := New()
	registry := core.NewPluginRegistry()
	if err := plugin.Register(registry); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	schemas := registry.Schemas()
	organismSchema, ok := schemas["organism"]
	if !ok {
		t.Fatalf("expected organism schema to be registered")
	}
	if organismSchema["$id"].(string) != "colonycore:frog:organism" {
		t.Fatalf("unexpected organism schema id: %v", organismSchema["$id"])
	}

	rules := registry.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected frog plugin to register one rule, got %d", len(rules))
	}

	datasets := registry.DatasetTemplates()
	if len(datasets) != 1 {
		t.Fatalf("expected one dataset template, got %d", len(datasets))
	}
	if datasets[0].Key != "frog_population_snapshot" {
		t.Fatalf("unexpected dataset key: %s", datasets[0].Key)
	}
	if datasets[0].Binder == nil {
		t.Fatalf("dataset binder should be registered")
	}
}

func TestFrogHabitatRuleOutcomes(t *testing.T) {
	svc := core.NewInMemoryService(core.NewRulesEngine())
	if _, err := svc.InstallPlugin(New()); err != nil {
		t.Fatalf("install frog plugin: %v", err)
	}
	ctx := context.Background()

	humid, _, err := svc.CreateHousingUnit(ctx, core.HousingUnit{Name: "Humid", Facility: "Lab", Capacity: 2, Environment: "humid"})
	if err != nil {
		t.Fatalf("create humid housing: %v", err)
	}
	dry, _, err := svc.CreateHousingUnit(ctx, core.HousingUnit{Name: "Dry", Facility: "Lab", Capacity: 2, Environment: "dry"})
	if err != nil {
		t.Fatalf("create dry housing: %v", err)
	}

	frogA, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Water", Species: "Tree Frog"})
	if err != nil {
		t.Fatalf("create frogA: %v", err)
	}
	frogB, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Desert", Species: "Poison Frog"})
	if err != nil {
		t.Fatalf("create frogB: %v", err)
	}
	other, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Lizard", Species: "Gecko"})
	if err != nil {
		t.Fatalf("create non-frog organism: %v", err)
	}

	if _, res, err := svc.AssignOrganismHousing(ctx, frogA.ID, humid.ID); err != nil {
		t.Fatalf("assign humid housing: %v", err)
	} else if len(res.Violations) != 0 {
		t.Fatalf("expected no violations for humid environment, got %+v", res.Violations)
	}

	if _, res, err := svc.AssignOrganismHousing(ctx, frogB.ID, dry.ID); err != nil {
		t.Fatalf("assign dry housing: %v", err)
	} else {
		if len(res.Violations) != 1 {
			t.Fatalf("expected one violation, got %+v", res.Violations)
		}
		if res.Violations[0].Rule != "frog_habitat_warning" {
			t.Fatalf("unexpected rule: %+v", res.Violations[0])
		}
		if res.Violations[0].Severity != core.SeverityWarn {
			t.Fatalf("expected warning severity")
		}
	}

	if _, res, err := svc.AssignOrganismHousing(ctx, other.ID, dry.ID); err != nil {
		t.Fatalf("assign non-frog housing: %v", err)
	} else {
		for _, violation := range res.Violations {
			if violation.EntityID == other.ID {
				t.Fatalf("unexpected violation for non-frog species: %+v", violation)
			}
		}
	}
}

func TestFrogPopulationDataset(t *testing.T) {
	svc := core.NewInMemoryService(core.NewRulesEngine())
	meta, err := svc.InstallPlugin(New())
	if err != nil {
		t.Fatalf("install frog plugin: %v", err)
	}
	if len(meta.Datasets) != 1 {
		t.Fatalf("expected dataset descriptor to be registered")
	}

	template, ok := svc.ResolveDatasetTemplate(meta.Datasets[0].Slug)
	if !ok {
		t.Fatalf("dataset template not resolved: %s", meta.Datasets[0].Slug)
	}

	ctx := context.Background()
	project, _, err := svc.CreateProject(ctx, core.Project{Code: "PRJ-FROG", Title: "Frog Ops"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	organism, _, err := svc.CreateOrganism(ctx, core.Organism{
		Name:      "Maple",
		Species:   "Tree Frog",
		Stage:     core.StageAdult,
		ProjectID: &project.ID,
	})
	if err != nil {
		t.Fatalf("create frog organism: %v", err)
	}
	if _, _, err := svc.CreateOrganism(ctx, core.Organism{Name: "Gecko", Species: "Gecko", Stage: core.StageAdult}); err != nil {
		t.Fatalf("create non frog organism: %v", err)
	}

	result, paramErrs, err := template.Run(ctx, map[string]any{"include_retired": true}, core.DatasetScope{ProjectIDs: []string{project.ID}}, core.FormatJSON)
	if err != nil {
		t.Fatalf("run dataset: %v", err)
	}
	if len(paramErrs) > 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected one row, got %d", len(result.Rows))
	}
	row := result.Rows[0]
	if row["organism_id"].(string) != organism.ID {
		t.Fatalf("unexpected organism in dataset: %+v", row)
	}
	if _, ok := result.Metadata["project_scope"]; !ok {
		t.Fatalf("expected project scope metadata")
	}

	filtered, _, err := template.Run(ctx, map[string]any{"stage": string(core.StageLarva)}, core.DatasetScope{}, core.FormatJSON)
	if err != nil {
		t.Fatalf("run dataset with stage filter: %v", err)
	}
	if len(filtered.Rows) != 0 {
		t.Fatalf("expected no larva rows, got %d", len(filtered.Rows))
	}
}

func TestFrogHabitatRuleName(t *testing.T) {
	if name := (frogHabitatRule{}).Name(); name == "" {
		t.Fatalf("expected frog habitat rule name")
	}
}

func TestFrogPopulationBinderFilters(t *testing.T) {
	svc := core.NewInMemoryService(core.NewDefaultRulesEngine())
	if _, err := svc.InstallPlugin(New()); err != nil {
		t.Fatalf("install frog plugin: %v", err)
	}
	ctx := context.Background()

	projectA, _, err := svc.CreateProject(ctx, core.Project{Code: "PRJ-A", Title: "Project A"})
	if err != nil {
		t.Fatalf("create project A: %v", err)
	}
	projectB, _, err := svc.CreateProject(ctx, core.Project{Code: "PRJ-B", Title: "Project B"})
	if err != nil {
		t.Fatalf("create project B: %v", err)
	}
	protocol, _, err := svc.CreateProtocol(ctx, core.Protocol{Code: "PROTO", Title: "Protocol", MaxSubjects: 10})
	if err != nil {
		t.Fatalf("create protocol: %v", err)
	}
	housing, _, err := svc.CreateHousingUnit(ctx, core.HousingUnit{Name: "Wet", Facility: "Lab", Capacity: 4, Environment: "humid"})
	if err != nil {
		t.Fatalf("create housing: %v", err)
	}
	housingID := housing.ID

	protocolID := protocol.ID
	projectAID := projectA.ID
	projectBID := projectB.ID

	frogA, _, err := svc.CreateOrganism(ctx, core.Organism{
		Name:       "Alpha",
		Species:    "Tree Frog",
		Stage:      core.StageAdult,
		ProjectID:  &projectAID,
		ProtocolID: &protocolID,
		HousingID:  &housingID,
	})
	if err != nil {
		t.Fatalf("create frog A: %v", err)
	}
	frogATime := frogA.UpdatedAt
	time.Sleep(5 * time.Millisecond)
	if _, _, err := svc.CreateOrganism(ctx, core.Organism{
		Name:       "Bravo",
		Species:    "Tree Frog",
		Stage:      core.StageAdult,
		ProjectID:  &projectBID,
		ProtocolID: &protocolID,
		HousingID:  &housingID,
	}); err != nil {
		t.Fatalf("create frog B: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, _, err := svc.CreateOrganism(ctx, core.Organism{
		Name:       "Charlie",
		Species:    "Tree Frog",
		Stage:      core.StageRetired,
		ProjectID:  &projectAID,
		ProtocolID: &protocolID,
	}); err != nil {
		t.Fatalf("create frog C: %v", err)
	}

	descriptor := svc.DatasetTemplates()[0]
	template, ok := svc.ResolveDatasetTemplate(descriptor.Slug)
	if !ok {
		t.Fatalf("resolve dataset template")
	}

	params := map[string]any{
		"as_of": frogATime.Add(time.Second).Format(time.RFC3339),
	}
	scope := core.DatasetScope{Requestor: "analyst", ProjectIDs: []string{projectAID}, ProtocolIDs: []string{protocolID}}
	result, paramErrs, err := template.Run(ctx, params, scope, core.FormatJSON)
	if err != nil {
		t.Fatalf("run dataset: %v", err)
	}
	if len(paramErrs) != 0 {
		t.Fatalf("unexpected parameter errors: %+v", paramErrs)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected single row after filters, got %d", len(result.Rows))
	}
	if result.Rows[0]["organism_id"].(string) != frogA.ID {
		t.Fatalf("expected frog A row, got %+v", result.Rows[0])
	}

	paramsRetired := map[string]any{
		"include_retired": true,
		"as_of":           time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	result, _, err = template.Run(ctx, paramsRetired, scope, core.FormatJSON)
	if err != nil {
		t.Fatalf("run dataset include retired: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected two rows with retired included, got %d", len(result.Rows))
	}

	blankScope := core.DatasetScope{ProjectIDs: []string{"nonexistent"}}
	result, _, err = template.Run(ctx, params, blankScope, core.FormatJSON)
	if err != nil {
		t.Fatalf("run dataset with mismatched scope: %v", err)
	}
	if len(result.Rows) != 0 {
		t.Fatalf("expected no rows for unmatched scope, got %d", len(result.Rows))
	}

	if contains([]string{"a", "b"}, "z") {
		t.Fatalf("expected contains helper to return false")
	}
	if value := valueOrNil(nil); value != nil {
		t.Fatalf("expected valueOrNil to return nil")
	}
}
