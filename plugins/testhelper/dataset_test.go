package testhelper

import (
	"testing"
	"time"
)

const mutatedValue = "mutated"

func TestOrganismFixtureBuilder(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	name := "Org1"
	lineID := "line-id"
	strainID := "strain-id"
	parentIDs := []string{"p1", "p2"}
	cfg := OrganismFixtureConfig{
		BaseFixture: BaseFixture{ID: "o1", CreatedAt: now, UpdatedAt: now},
		Name:        name, Species: "sp", Line: "L", LineID: &lineID, StrainID: &strainID, ParentIDs: parentIDs, Stage: LifecycleStages().Adult,
	}
	org := Organism(cfg)
	if org.Name() != name || org.Stage() != LifecycleStages().Adult {
		t.Fatalf("unexpected organism attributes: %s %v", org.Name(), org.Stage())
	}
	if got, ok := org.LineID(); !ok || got != lineID {
		t.Fatalf("expected line id %s, got %s", lineID, got)
	}
	if got, ok := org.StrainID(); !ok || got != strainID {
		t.Fatalf("expected strain id %s, got %s", strainID, got)
	}
	gotParents := org.ParentIDs()
	if len(gotParents) != len(parentIDs) || gotParents[0] != "p1" || gotParents[1] != "p2" {
		t.Fatalf("unexpected parent ids: %+v", gotParents)
	}
	list := Organisms(cfg)
	if len(list) != 1 || list[0].Name() != name {
		t.Fatalf("Organisms slice builder failed")
	}
}

func TestHousingUnitFixtureBuilder(t *testing.T) {
	now := time.Now().UTC()
	cfg := HousingUnitFixtureConfig{BaseFixture: BaseFixture{ID: "h1", CreatedAt: now, UpdatedAt: now}, Name: "H", FacilityID: "F", Capacity: 2, Environment: "terrestrial"}
	hu := HousingUnit(cfg)
	if hu.Capacity() != 2 || hu.Environment() != "terrestrial" {
		t.Fatalf("unexpected housing unit values")
	}
}

func TestOrganismHelperClonesData(t *testing.T) {
	now := time.Now().UTC()
	project := "project"
	housing := "housing"
	attributes := map[string]any{"flag": true}
	lineID := "line-id"
	parentIDs := []string{"p1"}

	organism := Organism(OrganismFixtureConfig{
		BaseFixture: BaseFixture{ID: "org", CreatedAt: now, UpdatedAt: now},
		Name:        "Name",
		Species:     "Species",
		Line:        "Line",
		LineID:      &lineID,
		ParentIDs:   parentIDs,
		Stage:       LifecycleStages().Adult,
		ProjectID:   &project,
		HousingID:   &housing,
		Attributes:  attributes,
	})

	if organism.ID() != "org" || organism.Name() != "Name" {
		t.Fatalf("unexpected organism values: %+v", organism)
	}
	if organism.Stage() != LifecycleStages().Adult {
		t.Fatalf("expected adult stage")
	}

	project = mutatedValue
	if value, _ := organism.ProjectID(); value != "project" {
		t.Fatalf("expected project pointer clone, got %s", value)
	}

	attrs := organism.Attributes()
	attrs["flag"] = false
	if organism.Attributes()["flag"] != true {
		t.Fatalf("expected attribute clone to remain immutable")
	}
	parentIDs[0] = mutatedValue
	if organism.ParentIDs()[0] != "p1" {
		t.Fatalf("expected parent id slice clone to remain unchanged")
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
		FacilityID:  "Facility",
		Capacity:    4,
		Environment: "humid",
	})

	if unit.ID() != "unit" || unit.Environment() != "humid" || unit.Capacity() != 4 {
		t.Fatalf("unexpected housing unit values: %+v", unit)
	}
}

func TestCloneHelpersCoverage(t *testing.T) {
	if cloneOptionalString(nil) != nil {
		t.Fatalf("expected nil optional string clone")
	}
	value := "orig"
	ptr := cloneOptionalString(&value)
	value = mutatedValue
	if ptr == nil || *ptr != "orig" {
		t.Fatalf("expected cloned string pointer to remain original")
	}

	attrs := map[string]any{
		"map":   map[string]any{"inner": []any{1, map[string]any{"k": "v"}}},
		"slice": []string{"a", "b"},
		"maps":  []map[string]any{{"flag": true}},
	}
	cloned := cloneAttributes(attrs)
	inner := cloned["map"].(map[string]any)["inner"].([]any)
	if len(inner) != 2 || inner[0].(int) != 1 {
		t.Fatalf("expected cloned inner slice to retain values")
	}
	if innerMap, ok := inner[1].(map[string]any); !ok || innerMap["k"].(string) != "v" {
		t.Fatalf("expected cloned nested map value")
	}
	attrs["slice"].([]string)[0] = mutatedValue
	if cloned["slice"].([]string)[0] != "a" {
		t.Fatalf("expected slice clone to remain unchanged")
	}
	attrs["maps"].([]map[string]any)[0]["flag"] = false
	if cloned["maps"].([]map[string]any)[0]["flag"].(bool) != true {
		t.Fatalf("expected map slice clone to remain true")
	}
	if out := cloneAttributes(nil); out != nil {
		t.Fatalf("expected nil attributes clone, got %+v", out)
	}
	if result := deepCloneAttr([]map[string]any{}); result == nil {
		t.Fatalf("expected empty slice clone")
	}
}
