package testhelper

import (
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
)

func TestOrganismHelperClonesData(t *testing.T) {
	now := time.Now().UTC()
	project := "project"
	housing := "housing"
	attributes := map[string]any{"flag": true}

	organism := Organism(OrganismFixtureConfig{
		BaseFixture: BaseFixture{ID: "org", CreatedAt: now, UpdatedAt: now},
		Name:        "Name",
		Species:     "Species",
		Line:        "Line",
		Stage:       datasetapi.StageAdult,
		ProjectID:   &project,
		HousingID:   &housing,
		Attributes:  attributes,
	})

	if organism.ID() != "org" || organism.Name() != "Name" {
		t.Fatalf("unexpected organism values: %+v", organism)
	}
	if organism.Stage() != datasetapi.StageAdult {
		t.Fatalf("expected adult stage")
	}

	project = "mutated"
	if value, _ := organism.ProjectID(); value != "project" {
		t.Fatalf("expected project pointer clone, got %s", value)
	}

	attrs := organism.Attributes()
	attrs["flag"] = false
	if organism.Attributes()["flag"] != true {
		t.Fatalf("expected attribute clone to remain immutable")
	}

	if got := Organisms(); got != nil {
		t.Fatalf("expected nil slice when no configs, got %+v", got)
	}
}

func TestHousingUnitHelper(t *testing.T) {
	now := time.Now().UTC()
	unit := HousingUnit(HousingUnitFixtureConfig{
		BaseFixture: BaseFixture{ID: "unit", CreatedAt: now, UpdatedAt: now},
		Name:        "Hab",
		Facility:    "Facility",
		Capacity:    4,
		Environment: "humid",
	})

	if unit.ID() != "unit" || unit.Environment() != "humid" || unit.Capacity() != 4 {
		t.Fatalf("unexpected housing unit values: %+v", unit)
	}
}
