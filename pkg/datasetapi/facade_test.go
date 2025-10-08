package datasetapi

import (
	"encoding/json"
	"testing"
	"time"
)

const (
	organismCohortID   = "cohort"
	organismHousingID  = "housing"
	organismProtocolID = "protocol"
	organismProjectID  = "project"
	mutatedLiteral     = "mutated"
)

func TestOrganismFacadeReadOnly(t *testing.T) {
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	cohort := organismCohortID
	housing := organismHousingID
	protocol := organismProtocolID
	project := organismProjectID
	attrs := map[string]any{"flag": true}

	organism := NewOrganism(OrganismData{
		Base:       BaseData{ID: "id", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Name:       "Alpha",
		Species:    "Frog",
		Line:       "Line",
		Stage:      LifecycleStage(NewLifecycleStageContext().Adult().String()),
		CohortID:   &cohort,
		HousingID:  &housing,
		ProtocolID: &protocol,
		ProjectID:  &project,
		Attributes: attrs,
	})

	expectedStage := LifecycleStage(NewLifecycleStageContext().Adult().String())
	if organism.ID() != "id" || organism.Name() != "Alpha" || organism.Stage() != expectedStage {
		t.Fatalf("unexpected organism values: %+v", organism)
	}
	if organism.Species() != "Frog" || organism.Line() != "Line" {
		t.Fatalf("unexpected organism species/line: %s %s", organism.Species(), organism.Line())
	}
	if organism.CreatedAt() != createdAt || organism.UpdatedAt() != updatedAt {
		t.Fatalf("unexpected timestamps: %+v", organism)
	}

	for _, check := range []struct {
		getter func() (string, bool)
		label  string
		expect string
	}{
		{organism.CohortID, "cohort", cohort},
		{organism.HousingID, "housing", housing},
		{organism.ProtocolID, "protocol", protocol},
		{organism.ProjectID, "project", project},
	} {
		value, ok := check.getter()
		if !ok || value != check.expect {
			t.Fatalf("expected %s to equal %s, got %q (ok=%v)", check.label, check.expect, value, ok)
		}
	}

	// Mutate original pointers to ensure the facade captured copies.
	cohort = mutatedLiteral
	housing = mutatedLiteral
	protocol = mutatedLiteral
	project = mutatedLiteral
	if value, _ := organism.CohortID(); value != organismCohortID {
		t.Fatalf("expected cohort clone to remain stable, got %s", value)
	}
	if value, _ := organism.HousingID(); value != organismHousingID {
		t.Fatalf("expected housing clone to remain stable, got %s", value)
	}

	attrsCopy := organism.Attributes()
	attrsCopy["flag"] = false
	if organism.Attributes()["flag"] != true {
		t.Fatalf("expected attributes clone to remain immutable")
	}

	payload, err := json.Marshal(organism)
	if err != nil {
		t.Fatalf("marshal organism: %v", err)
	}
	var serialized map[string]any
	if err := json.Unmarshal(payload, &serialized); err != nil {
		t.Fatalf("unmarshal organism json: %v", err)
	}
	if serialized["id"] != "id" || serialized["species"] != "Frog" {
		t.Fatalf("unexpected serialized organism: %+v", serialized)
	}
}

func TestBreedingProcedureFacadesCloneCollections(t *testing.T) {
	created := time.Now().UTC()
	housing := organismHousingID
	protocol := organismProtocolID
	breeding := NewBreedingUnit(BreedingUnitData{
		Base:       BaseData{ID: "breed", CreatedAt: created},
		Name:       "Breeding",
		Strategy:   "pair",
		HousingID:  &housing,
		ProtocolID: &protocol,
		FemaleIDs:  []string{"f1"},
		MaleIDs:    []string{"m1"},
	})

	females := breeding.FemaleIDs()
	females[0] = mutatedLiteral
	if breeding.Name() != "Breeding" || breeding.FemaleIDs()[0] != "f1" {
		t.Fatalf("expected female ids clone to remain unchanged")
	}
	if id, ok := breeding.HousingID(); !ok || id != housing {
		t.Fatalf("expected housing id cloned, got %q", id)
	}
	housing = mutatedLiteral
	if id, _ := breeding.HousingID(); id != organismHousingID {
		t.Fatalf("expected housing id clone to be stable")
	}

	procedure := NewProcedure(ProcedureData{
		Base:        BaseData{ID: "proc", UpdatedAt: created},
		Name:        "Procedure",
		Status:      "scheduled",
		ScheduledAt: created.Add(time.Hour),
		ProtocolID:  organismProtocolID,
		CohortID:    &housing,
		OrganismIDs: []string{"o1"},
	})

	ids := procedure.OrganismIDs()
	ids[0] = mutatedLiteral
	if procedure.OrganismIDs()[0] != "o1" {
		t.Fatalf("expected organism ids clone to remain unchanged")
	}
	if _, ok := procedure.CohortID(); !ok {
		t.Fatalf("expected cohort id to be present")
	}

	if payload, err := json.Marshal(breeding); err != nil {
		t.Fatalf("marshal breeding: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal breeding: %v", err)
		}
		if serialized["id"] != "breed" || serialized["strategy"] != "pair" {
			t.Fatalf("unexpected breeding json: %+v", serialized)
		}
	}

	if payload, err := json.Marshal(procedure); err != nil {
		t.Fatalf("marshal procedure: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal procedure: %v", err)
		}
		if serialized["id"] != "proc" || serialized["protocol_id"] != "protocol" {
			t.Fatalf("unexpected procedure json: %+v", serialized)
		}
	}
}

func TestProtocolProjectCohortFacades(t *testing.T) {
	now := time.Now().UTC()
	cohortProject := "project"
	cohortHousing := "housing"
	cohortProtocol := "protocol"

	cohort := NewCohort(CohortData{
		Base:       BaseData{ID: "cohort", CreatedAt: now},
		Name:       "Cohort",
		Purpose:    "Study",
		ProjectID:  &cohortProject,
		HousingID:  &cohortHousing,
		ProtocolID: &cohortProtocol,
	})
	if cohort.Name() != "Cohort" || cohort.Purpose() != "Study" {
		t.Fatalf("unexpected cohort values")
	}

	protocol := NewProtocol(ProtocolData{
		Base:        BaseData{ID: "protocol", CreatedAt: now},
		Code:        "P",
		Title:       "Protocol",
		Description: "Desc",
		MaxSubjects: 5,
	})
	if protocol.Code() != "P" || protocol.Title() != "Protocol" || protocol.MaxSubjects() != 5 {
		t.Fatalf("unexpected protocol values")
	}

	project := NewProject(ProjectData{
		Base:        BaseData{ID: "project", CreatedAt: now},
		Code:        "PR",
		Title:       "Project",
		Description: "Description",
	})
	if project.Code() != "PR" || project.Title() != "Project" || project.Description() != "Description" {
		t.Fatalf("unexpected project values")
	}

	housing := NewHousingUnit(HousingUnitData{
		Base:        BaseData{ID: "housing", CreatedAt: now},
		Name:        "Habitat",
		Facility:    "Facility",
		Capacity:    3,
		Environment: "humid",
	})
	if housing.Environment() != "humid" || housing.Capacity() != 3 {
		t.Fatalf("unexpected housing values")
	}

	// Mutate original pointers to ensure cohort retains clones.
	cohortProject = "changed"
	if value, _ := cohort.ProjectID(); value != "project" {
		t.Fatalf("expected cohort project id to remain unchanged")
	}

	if payload, err := json.Marshal(cohort); err != nil {
		t.Fatalf("marshal cohort: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal cohort: %v", err)
		}
		if serialized["id"] != "cohort" || serialized["purpose"] != "Study" {
			t.Fatalf("unexpected cohort json: %+v", serialized)
		}
	}

	if payload, err := json.Marshal(protocol); err != nil {
		t.Fatalf("marshal protocol: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal protocol: %v", err)
		}
		if serialized["code"] != "P" || serialized["max_subjects"] != float64(5) {
			t.Fatalf("unexpected protocol json: %+v", serialized)
		}
	}

	if payload, err := json.Marshal(project); err != nil {
		t.Fatalf("marshal project: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal project: %v", err)
		}
		if serialized["code"] != "PR" || serialized["title"] != "Project" {
			t.Fatalf("unexpected project json: %+v", serialized)
		}
	}

	if payload, err := json.Marshal(housing); err != nil {
		t.Fatalf("marshal housing: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal housing: %v", err)
		}
		if serialized["id"] != "housing" || serialized["environment"] != "humid" {
			t.Fatalf("unexpected housing json: %+v", serialized)
		}
	}
}

func TestOptionalAccessorsBehaviors(t *testing.T) {
	organism := NewOrganism(OrganismData{Base: BaseData{ID: "noopts"}})
	if _, ok := organism.CohortID(); ok {
		t.Fatalf("expected cohort accessor to report missing value")
	}
	if _, ok := organism.HousingID(); ok {
		t.Fatalf("expected housing accessor to report missing value")
	}
	if _, ok := organism.ProtocolID(); ok {
		t.Fatalf("expected protocol accessor to report missing value")
	}
	if _, ok := organism.ProjectID(); ok {
		t.Fatalf("expected project accessor to report missing value")
	}
	if organism.Attributes() != nil {
		t.Fatalf("expected nil attributes for empty map")
	}

	breeding := NewBreedingUnit(BreedingUnitData{Base: BaseData{ID: "breed"}})
	if breeding.FemaleIDs() != nil || breeding.MaleIDs() != nil {
		t.Fatalf("expected female/male slices to be nil when empty")
	}

	procedure := NewProcedure(ProcedureData{Base: BaseData{ID: "proc"}, ProtocolID: "proto"})
	if _, ok := procedure.CohortID(); ok {
		t.Fatalf("expected cohort accessor on procedure to report missing value")
	}
}

func TestDeepCloneAttributes(t *testing.T) {
	nested := map[string]any{
		"level1": map[string]any{
			"level2": []any{map[string]any{"k": "v"}, []string{"a", "b"}},
		},
	}
	org := NewOrganism(OrganismData{Base: BaseData{ID: "deep"}, Attributes: nested})
	attrs := org.Attributes()
	// mutate returned structure deeply
	lvl1 := attrs["level1"].(map[string]any)
	lvl2 := lvl1["level2"].([]any)
	innerMap := lvl2[0].(map[string]any)
	innerMap["k"] = "mutated"
	slice := lvl2[1].([]string)
	slice[0] = "z"
	// original should remain unchanged
	orig := org.Attributes()
	oLvl1 := orig["level1"].(map[string]any)
	oLvl2 := oLvl1["level2"].([]any)
	oInnerMap := oLvl2[0].(map[string]any)
	if oInnerMap["k"].(string) != "v" {
		t.Fatalf("expected deep-cloned inner map value 'v', got %v", oInnerMap["k"])
	}
	oSlice := oLvl2[1].([]string)
	if oSlice[0] != "a" {
		t.Fatalf("expected deep-cloned string slice element 'a', got %s", oSlice[0])
	}
}

func TestOrganismContextualMethods(t *testing.T) {
	createdAt := time.Now().UTC()
	updatedAt := createdAt.Add(time.Hour)

	stages := NewLifecycleStageContext()

	// Test different lifecycle stages
	testCases := []struct {
		name             string
		stage            LifecycleStage
		expectedActive   bool
		expectedRetired  bool
		expectedDeceased bool
	}{
		{"adult", LifecycleStage(stages.Adult().String()), true, false, false},
		{"juvenile", LifecycleStage(stages.Juvenile().String()), true, false, false},
		{"larva", LifecycleStage(stages.Larva().String()), true, false, false},
		{"planned", LifecycleStage(stages.Planned().String()), true, false, false},
		{"retired", LifecycleStage(stages.Retired().String()), false, true, false},
		{"deceased", LifecycleStage(stages.Deceased().String()), false, false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			organism := NewOrganism(OrganismData{
				Base:    BaseData{ID: "test", CreatedAt: createdAt, UpdatedAt: updatedAt},
				Name:    "Test",
				Species: "Test Species",
				Line:    "Test Line",
				Stage:   tc.stage,
			})

			// Test GetCurrentStage
			currentStage := organism.GetCurrentStage()
			if currentStage.String() != string(tc.stage) {
				t.Errorf("expected GetCurrentStage().String() to be '%s', got '%s'", tc.stage, currentStage.String())
			}

			// Test IsActive
			if organism.IsActive() != tc.expectedActive {
				t.Errorf("expected IsActive() to be %t for stage %s, got %t", tc.expectedActive, tc.stage, organism.IsActive())
			}

			// Test IsRetired
			if organism.IsRetired() != tc.expectedRetired {
				t.Errorf("expected IsRetired() to be %t for stage %s, got %t", tc.expectedRetired, tc.stage, organism.IsRetired())
			}

			// Test IsDeceased
			if organism.IsDeceased() != tc.expectedDeceased {
				t.Errorf("expected IsDeceased() to be %t for stage %s, got %t", tc.expectedDeceased, tc.stage, organism.IsDeceased())
			}
		})
	}
}

func TestOrganismGetCurrentStageEdgeCases(t *testing.T) {
	// Test unknown stage fallback
	organism := NewOrganism(OrganismData{
		Base:    BaseData{ID: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:    "Test",
		Species: "Test Species",
		Line:    "Test Line",
		Stage:   LifecycleStage("unknown_stage"),
	})

	currentStage := organism.GetCurrentStage()
	stages := NewLifecycleStageContext()
	if !currentStage.Equals(stages.Adult()) {
		t.Error("expected unknown stage to fallback to Adult")
	}
}
