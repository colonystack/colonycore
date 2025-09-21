package frog

import (
	"context"
	"testing"

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
