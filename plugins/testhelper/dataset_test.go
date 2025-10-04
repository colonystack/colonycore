package testhelper

import (
	"colonycore/pkg/datasetapi"
	"testing"
	"time"
)

func TestOrganismFixtureBuilder(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	name := "Org1"
	cfg := OrganismFixtureConfig{
		BaseFixture: BaseFixture{ID: "o1", CreatedAt: now, UpdatedAt: now},
		Name:        name, Species: "sp", Line: "L", Stage: datasetapi.StageAdult,
	}
	org := Organism(cfg)
	if org.Name() != name || org.Stage() != datasetapi.StageAdult {
		t.Fatalf("unexpected organism attributes: %s %v", org.Name(), org.Stage())
	}
	list := Organisms(cfg)
	if len(list) != 1 || list[0].Name() != name {
		t.Fatalf("Organisms slice builder failed")
	}
}

func TestHousingUnitFixtureBuilder(t *testing.T) {
	now := time.Now().UTC()
	cfg := HousingUnitFixtureConfig{BaseFixture: BaseFixture{ID: "h1", CreatedAt: now, UpdatedAt: now}, Name: "H", Facility: "F", Capacity: 2, Environment: "env"}
	hu := HousingUnit(cfg)
	if hu.Capacity() != 2 || hu.Environment() != "env" {
		t.Fatalf("unexpected housing unit values")
	}
}

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
