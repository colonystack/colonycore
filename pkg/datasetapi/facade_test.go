package datasetapi

import (
	"encoding/json"
	"testing"
	"time"
)

const (
	organismCohortID      = "cohort"
	organismHousingID     = "housing"
	organismProtocolID    = "protocol"
	organismProjectID     = "project"
	mutatedLiteral        = "mutated"
	pairingPurposeLineage = "lineage"
	coreAttributeValue    = "value"
)

func strPtr(v string) *string {
	return &v
}

func TestOrganismFacadeReadOnly(t *testing.T) {
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	hooks := NewExtensionHookContext()
	contributors := NewExtensionContributorContext()
	orgHook := hooks.OrganismAttributes()
	corePlugin := contributors.Core()
	cohort := organismCohortID
	housing := organismHousingID
	protocol := organismProtocolID
	project := organismProjectID
	attrs := map[string]any{"flag": true}
	lineID := "line-id"
	strainID := "strain-id"
	parentIDs := []string{"p1", "p2"}

	organism := NewOrganism(OrganismData{
		Base:       BaseData{ID: "id", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Name:       "Alpha",
		Species:    "Frog",
		Line:       "Line",
		LineID:     &lineID,
		StrainID:   &strainID,
		ParentIDs:  parentIDs,
		Stage:      LifecycleStage(NewLifecycleStageContext().Adult().String()),
		CohortID:   &cohort,
		HousingID:  &housing,
		ProtocolID: &protocol,
		ProjectID:  &project,
		Extensions: newCoreExtensionSet(orgHook, attrs),
	})

	expectedStage := LifecycleStage(NewLifecycleStageContext().Adult().String())
	if organism.ID() != "id" || organism.Name() != "Alpha" || organism.Stage() != expectedStage {
		t.Fatalf("unexpected organism values: %+v", organism)
	}
	if organism.Species() != "Frog" || organism.Line() != "Line" {
		t.Fatalf("unexpected organism species/line: %s %s", organism.Species(), organism.Line())
	}
	if got, ok := organism.LineID(); !ok || got != lineID {
		t.Fatalf("expected line id %s, got %s", lineID, got)
	}
	if got, ok := organism.StrainID(); !ok || got != strainID {
		t.Fatalf("expected strain id %s, got %s", strainID, got)
	}
	if p := organism.ParentIDs(); len(p) != len(parentIDs) || p[0] != "p1" {
		t.Fatalf("unexpected parent ids: %+v", p)
	}
	if organism.CreatedAt() != createdAt || organism.UpdatedAt() != updatedAt {
		t.Fatalf("unexpected timestamps: %+v", organism)
	}

	for _, check := range []struct {
		getter func() (string, bool)
		label  string
		expect string
	}{
		{organism.CohortID, organismCohortID, cohort},
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
	parentIDs[0] = mutatedLiteral
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
	extSet := organism.Extensions()
	if plugins := extSet.Plugins(orgHook); len(plugins) != 1 || !plugins[0].Equals(corePlugin) {
		t.Fatalf("expected core plugin payload, got %+v", plugins)
	}
	corePayload, ok := extSet.Core(orgHook)
	if !ok {
		t.Fatal("expected organism extension payload")
	}
	corePayload.(map[string]any)["flag"] = false
	if fresh, _ := extSet.Core(orgHook); fresh.(map[string]any)["flag"] != true {
		t.Fatalf("expected extension payload clone to remain stable, got %+v", fresh)
	}
	raw := extSet.Raw()
	raw[orgHook.value()][corePlugin.value()].(map[string]any)["flag"] = false
	if fresh, _ := extSet.Core(orgHook); fresh.(map[string]any)["flag"] != true {
		t.Fatalf("expected raw mutation not to leak into stored payload")
	}
	if organism.ParentIDs()[0] != "p1" {
		t.Fatalf("expected parent ids clone to remain immutable")
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
	if serialized["line_id"] != "line-id" {
		t.Fatalf("expected serialized line id, got %+v", serialized["line_id"])
	}
	if serialized["strain_id"] != "strain-id" {
		t.Fatalf("expected serialized strain id, got %+v", serialized["strain_id"])
	}
	if parents, ok := serialized["parent_ids"].([]any); !ok || len(parents) != 2 {
		t.Fatalf("expected serialized parent ids")
	}
}

func TestBreedingProcedureFacadesCloneCollections(t *testing.T) {
	created := time.Now().UTC()
	housing := organismHousingID
	protocol := organismProtocolID
	lineID := "line-1"
	lineSnapshot := lineID
	strainID := "strain-1"
	strainSnapshot := strainID
	targetLineID := "line-2"
	targetLineSnapshot := targetLineID
	targetStrainID := "strain-2"
	targetStrainSnapshot := targetStrainID
	hookCtx := NewExtensionHookContext()
	breedingHook := hookCtx.BreedingUnitPairingAttributes()
	breeding := NewBreedingUnit(BreedingUnitData{
		Base:           BaseData{ID: "breed", CreatedAt: created},
		Name:           "Breeding",
		Strategy:       "pair",
		HousingID:      &housing,
		ProtocolID:     &protocol,
		LineID:         &lineID,
		StrainID:       &strainID,
		TargetLineID:   &targetLineID,
		TargetStrainID: &targetStrainID,
		PairingIntent:  strPtr("outcross"),
		PairingNotes:   strPtr("Documented pairing intent"),
		Extensions:     newCoreExtensionSet(breedingHook, map[string]any{"purpose": pairingPurposeLineage}),
		FemaleIDs:      []string{"f1"},
		MaleIDs:        []string{"m1"},
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
	if got, ok := breeding.LineID(); !ok || got != lineSnapshot {
		t.Fatalf("expected line id clone, got %q", got)
	}
	lineID = mutatedLiteral
	if got, _ := breeding.LineID(); got != lineSnapshot {
		t.Fatalf("expected line id clone to remain stable, got %q", got)
	}
	if got, ok := breeding.StrainID(); !ok || got != strainSnapshot {
		t.Fatalf("expected strain id clone, got %q", got)
	}
	strainID = mutatedLiteral
	if got, _ := breeding.StrainID(); got != strainSnapshot {
		t.Fatalf("expected strain id clone to remain stable, got %q", got)
	}
	if got, ok := breeding.TargetLineID(); !ok || got != targetLineSnapshot {
		t.Fatalf("expected target line id clone, got %q", got)
	}
	targetLineID = mutatedLiteral
	if got, _ := breeding.TargetLineID(); got != targetLineSnapshot {
		t.Fatalf("expected target line id clone to remain stable, got %q", got)
	}
	if got, ok := breeding.TargetStrainID(); !ok || got != targetStrainSnapshot {
		t.Fatalf("expected target strain id clone, got %q", got)
	}
	targetStrainID = mutatedLiteral
	if got, _ := breeding.TargetStrainID(); got != targetStrainSnapshot {
		t.Fatalf("expected target strain id clone to remain stable, got %q", got)
	}
	if intent := breeding.PairingIntent(); intent != "outcross" {
		t.Fatalf("expected pairing intent clone, got %q", intent)
	}
	if notes := breeding.PairingNotes(); notes != "Documented pairing intent" {
		t.Fatalf("expected pairing notes clone, got %q", notes)
	}
	attrs := breeding.PairingAttributes()
	attrs["purpose"] = mutatedLiteral
	if breeding.PairingAttributes()["purpose"] != pairingPurposeLineage {
		t.Fatalf("expected pairing attributes map to be cloned")
	}
	ext := breeding.Extensions()
	if payload, ok := ext.Core(breedingHook); !ok {
		t.Fatal("expected breeding unit extension payload")
	} else {
		payload.(map[string]any)["purpose"] = mutatedLiteral
		if fresh, _ := ext.Core(breedingHook); fresh.(map[string]any)["purpose"] != pairingPurposeLineage {
			t.Fatalf("expected breeding unit extension payload to be cloned, got %+v", fresh)
		}
	}

	procedureProject := "project-1"
	ProcedureProjectClone := procedureProject
	treatmentIDs := []string{"t1"}
	observationIDs := []string{"obs1"}
	procedure := NewProcedure(ProcedureData{
		Base:           BaseData{ID: "proc", UpdatedAt: created},
		Name:           "Procedure",
		Status:         "scheduled",
		ScheduledAt:    created.Add(time.Hour),
		ProtocolID:     organismProtocolID,
		ProjectID:      &procedureProject,
		CohortID:       &housing,
		OrganismIDs:    []string{"o1"},
		TreatmentIDs:   append([]string(nil), treatmentIDs...),
		ObservationIDs: append([]string(nil), observationIDs...),
	})

	ids := procedure.OrganismIDs()
	ids[0] = mutatedLiteral
	if procedure.OrganismIDs()[0] != "o1" {
		t.Fatalf("expected organism ids clone to remain unchanged")
	}
	if _, ok := procedure.CohortID(); !ok {
		t.Fatalf("expected cohort id to be present")
	}
	if projectID, ok := procedure.ProjectID(); !ok || projectID != ProcedureProjectClone {
		t.Fatalf("expected project id clone to be present")
	}
	procedureProject = mutatedLiteral
	if projectID, _ := procedure.ProjectID(); projectID != ProcedureProjectClone {
		t.Fatalf("expected project id clone to remain stable, got %q", projectID)
	}
	treatments := procedure.TreatmentIDs()
	if len(treatments) != 1 || treatments[0] != "t1" {
		t.Fatalf("expected treatment ids clone, got %+v", treatments)
	}
	treatments[0] = mutatedLiteral
	if procedure.TreatmentIDs()[0] != "t1" {
		t.Fatalf("expected treatment ids clone to remain stable")
	}
	observationIDs[0] = mutatedLiteral
	if procedure.ObservationIDs()[0] != "obs1" {
		t.Fatalf("expected observation ids clone to remain stable")
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
		if serialized["line_id"] != lineSnapshot {
			t.Fatalf("expected serialized line id, got %+v", serialized["line_id"])
		}
		if serialized["pairing_intent"] != "outcross" {
			t.Fatalf("expected serialized pairing intent, got %+v", serialized["pairing_intent"])
		}
		if attrs, ok := serialized["pairing_attributes"].(map[string]any); !ok || attrs["purpose"] != pairingPurposeLineage {
			t.Fatalf("expected serialized pairing attributes, got %+v", serialized["pairing_attributes"])
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
		if serialized["project_id"] != ProcedureProjectClone {
			t.Fatalf("expected serialized project id, got %+v", serialized["project_id"])
		}
		if ids, ok := serialized["treatment_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "t1" {
			t.Fatalf("expected serialized treatment ids, got %+v", serialized["treatment_ids"])
		}
		if ids, ok := serialized["observation_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "obs1" {
			t.Fatalf("expected serialized observation ids, got %+v", serialized["observation_ids"])
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
		Description: strPtr("Desc"),
		MaxSubjects: 5,
	})
	if protocol.Code() != "P" || protocol.Title() != "Protocol" || protocol.MaxSubjects() != 5 {
		t.Fatalf("unexpected protocol values")
	}

	projectFacilityIDs := []string{"facility-1"}
	projectProtocolIDs := []string{"protocol-1"}
	projectOrganismIDs := []string{"organism-1"}
	projectProcedureIDs := []string{"procedure-1"}
	projectSupplyIDs := []string{"supply-1"}
	project := NewProject(ProjectData{
		Base:          BaseData{ID: "project", CreatedAt: now},
		Code:          "PR",
		Title:         "Project",
		Description:   strPtr("Description"),
		FacilityIDs:   append([]string(nil), projectFacilityIDs...),
		ProtocolIDs:   append([]string(nil), projectProtocolIDs...),
		OrganismIDs:   append([]string(nil), projectOrganismIDs...),
		ProcedureIDs:  append([]string(nil), projectProcedureIDs...),
		SupplyItemIDs: append([]string(nil), projectSupplyIDs...),
	})
	if project.Code() != "PR" || project.Title() != "Project" || project.Description() != "Description" {
		t.Fatalf("unexpected project values")
	}
	if len(project.FacilityIDs()) != 1 || len(project.ProtocolIDs()) != 1 || len(project.OrganismIDs()) != 1 || len(project.ProcedureIDs()) != 1 || len(project.SupplyItemIDs()) != 1 {
		t.Fatalf("expected project relationships to round-trip")
	}
	projectProtocolIDs[0] = mutatedLiteral
	projectOrganismIDs[0] = mutatedLiteral
	projectProcedureIDs[0] = mutatedLiteral
	projectSupplyIDs[0] = mutatedLiteral
	if project.ProtocolIDs()[0] != "protocol-1" || project.OrganismIDs()[0] != "organism-1" || project.ProcedureIDs()[0] != "procedure-1" || project.SupplyItemIDs()[0] != "supply-1" {
		t.Fatalf("expected project relationship slices to be cloned")
	}

	housing := NewHousingUnit(HousingUnitData{
		Base:        BaseData{ID: "housing", CreatedAt: now},
		Name:        "Habitat",
		FacilityID:  "Facility",
		Capacity:    3,
		Environment: "humid",
	})
	if housing.Environment() != envHumid || housing.Capacity() != 3 {
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
		if serialized["id"] != organismCohortID || serialized["purpose"] != "Study" {
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
		if ids, ok := serialized["facility_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "facility-1" {
			t.Fatalf("expected facility ids in serialized project, got %+v", serialized["facility_ids"])
		}
		if ids, ok := serialized["protocol_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "protocol-1" {
			t.Fatalf("expected protocol ids in serialized project, got %+v", serialized["protocol_ids"])
		}
		if ids, ok := serialized["organism_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "organism-1" {
			t.Fatalf("expected organism ids in serialized project, got %+v", serialized["organism_ids"])
		}
		if ids, ok := serialized["procedure_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "procedure-1" {
			t.Fatalf("expected procedure ids in serialized project, got %+v", serialized["procedure_ids"])
		}
		if ids, ok := serialized["supply_item_ids"].([]any); !ok || len(ids) != 1 || ids[0] != "supply-1" {
			t.Fatalf("expected supply item ids in serialized project, got %+v", serialized["supply_item_ids"])
		}
	}

	if payload, err := json.Marshal(housing); err != nil {
		t.Fatalf("marshal housing: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal housing: %v", err)
		}
		if serialized["id"] != "housing" || serialized["environment"] != envHumid {
			t.Fatalf("unexpected housing json: %+v", serialized)
		}
	}
}

//nolint:gocyclo // This comprehensive facade test covers many entity types and has inherent complexity
func TestExtendedFacades(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	housingID := "H1"
	hooks := NewExtensionHookContext()

	facility := NewFacility(FacilityData{
		Base:           BaseData{ID: "facility", CreatedAt: now, UpdatedAt: now},
		Code:           "FAC-1",
		Name:           "Biosecure",
		Zone:           "Biosecure Wing",
		AccessPolicy:   "Restricted",
		Extensions:     newCoreExtensionSet(hooks.FacilityEnvironmentBaselines(), map[string]any{"temp": 21}),
		HousingUnitIDs: []string{housingID},
		ProjectIDs:     []string{"proj"},
	})
	if facility.Name() == "" || facility.Zone() == "" || facility.AccessPolicy() == "" {
		t.Fatal("facility getters should expose values")
	}
	if _, ok := facility.EnvironmentBaselines()["temp"]; !ok {
		t.Fatal("facility baselines should round-trip")
	}
	if len(facility.HousingUnitIDs()) != 1 || len(facility.ProjectIDs()) != 1 {
		t.Fatal("facility should expose related ids")
	}
	if !facility.SupportsHousingUnit(housingID) {
		t.Fatal("facility should support housing id")
	}
	if facility.Code() != "FAC-1" {
		t.Fatalf("expected facility code in facade, got %q", facility.Code())
	}
	facilityExt := facility.Extensions()
	facilityHook := hooks.FacilityEnvironmentBaselines()
	if payload, ok := facilityExt.Core(facilityHook); !ok {
		t.Fatal("expected facility extension payload")
	} else {
		payload.(map[string]any)["temp"] = 30
		if facility.EnvironmentBaselines()["temp"] != 21 {
			t.Fatal("expected facility baselines clone to remain stable after extension mutation")
		}
	}
	if !facility.GetZone().IsBiosecure() || !facility.GetAccessPolicy().IsRestricted() {
		t.Fatal("facility contextual helpers should reflect semantics")
	}
	if payload, err := json.Marshal(facility); err != nil {
		t.Fatalf("marshal facility: %v", err)
	} else {
		var serialized map[string]any
		if err := json.Unmarshal(payload, &serialized); err != nil {
			t.Fatalf("unmarshal facility: %v", err)
		}
		if serialized["code"] != "FAC-1" {
			t.Fatalf("expected facility code in json, got %+v", serialized)
		}
		if serialized["name"] != "Biosecure" || serialized["zone"] != "Biosecure Wing" {
			t.Fatalf("unexpected facility json: %+v", serialized)
		}
	}

	treatment := NewTreatment(TreatmentData{
		Base:              BaseData{ID: "treatment", CreatedAt: now},
		Name:              "Dose A",
		ProcedureID:       "proc",
		OrganismIDs:       []string{"org"},
		CohortIDs:         []string{"cohort"},
		DosagePlan:        "dose plan",
		AdministrationLog: []string{"dose"},
		AdverseEvents:     []string{"note"},
	})
	if treatment.Name() == "" || treatment.DosagePlan() == "" {
		t.Fatal("treatment facade should expose fields")
	}
	if len(treatment.OrganismIDs()) != 1 || len(treatment.CohortIDs()) != 1 {
		t.Fatal("treatment should expose related ids")
	}
	if !treatment.HasAdverseEvents() || !treatment.IsCompleted() {
		t.Fatal("treatment helpers should reflect log state")
	}
	if len(treatment.AdministrationLog()) != 1 || len(treatment.AdverseEvents()) != 1 {
		t.Fatal("treatment should expose administration log and adverse events")
	}

	procID := "proc"
	organObs := "org-obs"
	cohortObs := "cohort-obs"
	observation := NewObservation(ObservationData{
		Base:        BaseData{ID: "observation", CreatedAt: now},
		RecordedAt:  now,
		Observer:    "tech",
		ProcedureID: &procID,
		OrganismID:  &organObs,
		CohortID:    &cohortObs,
		Notes:       strPtr(coreAttributeValue),
		Extensions:  newCoreExtensionSet(hooks.ObservationData(), map[string]any{"score": 1}),
	})
	if observation.Observer() == "" || observation.Notes() == "" {
		t.Fatal("observation should retain data")
	}
	if val, ok := observation.ProcedureID(); !ok || val != procID {
		t.Fatalf("expected observation procedure id %q, got %q (ok=%v)", procID, val, ok)
	}
	if val, ok := observation.OrganismID(); !ok || val != organObs {
		t.Fatalf("expected observation organism id %q, got %q (ok=%v)", organObs, val, ok)
	}
	if val, ok := observation.CohortID(); !ok || val != cohortObs {
		t.Fatalf("expected observation cohort id %q, got %q (ok=%v)", cohortObs, val, ok)
	}
	if !observation.RecordedAt().Equal(now) {
		t.Fatalf("expected recorded at %v, got %v", now, observation.RecordedAt())
	}
	if _, ok := observation.ProcedureID(); !ok {
		t.Fatal("observation should expose procedure id")
	}
	shape := observation.GetDataShape()
	if !shape.HasStructuredPayload() || !shape.HasNarrativeNotes() {
		t.Fatal("observation mixed data shape should report both semantics")
	}
	if observation.Data()["score"] != 1 || observation.Notes() == "" {
		t.Fatal("observation should expose data payload and notes")
	}
	if payload, ok := observation.Extensions().Core(hooks.ObservationData()); !ok {
		t.Fatal("expected observation extension payload")
	} else {
		payload.(map[string]any)["score"] = 99
		if observation.Data()["score"] != 1 {
			t.Fatal("expected observation data clone to remain stable after extension mutation")
		}
	}

	organID := "org"
	cohortSample := "cohort"
	sample := NewSample(SampleData{
		Base:            BaseData{ID: "sample", CreatedAt: now},
		Identifier:      "S1",
		SourceType:      "organism",
		OrganismID:      &organID,
		CohortID:        &cohortSample,
		FacilityID:      facility.ID(),
		CollectedAt:     now,
		Status:          "stored",
		StorageLocation: "freezer",
		AssayType:       "assay",
		Extensions:      newCoreExtensionSet(hooks.SampleAttributes(), map[string]any{"key": coreAttributeValue}),
		ChainOfCustody: []SampleCustodyEventData{{
			Actor:     "tech",
			Location:  "lab",
			Timestamp: now,
			Notes:     strPtr("handoff"),
		}},
	})
	if sample.Identifier() == "" || sample.AssayType() == "" || sample.StorageLocation() == "" {
		t.Fatal("sample should expose fields")
	}
	if _, ok := sample.OrganismID(); !ok {
		t.Fatal("sample should expose organism id")
	}
	if val, ok := sample.CohortID(); !ok || val != cohortSample {
		t.Fatalf("expected cohort value %q, got %q (ok=%v)", cohortSample, val, ok)
	}
	if len(sample.ChainOfCustody()) != 1 {
		t.Fatal("sample custody events should be preserved")
	}
	event := sample.ChainOfCustody()[0]
	if event.Actor() != "tech" || event.Location() != "lab" || event.Notes() != "handoff" {
		t.Fatalf("unexpected custody event contents: %+v", event)
	}
	if !event.Timestamp().Equal(now) {
		t.Fatalf("expected custody timestamp %v, got %v", now, event.Timestamp())
	}
	if !sample.IsAvailable() || !sample.GetSource().IsOrganismDerived() {
		t.Fatal("sample helpers should report availability/source")
	}
	if sample.Status() == "" || sample.StorageLocation() == "" {
		t.Fatal("sample should expose status and storage location")
	}
	if sample.Attributes()["key"] != coreAttributeValue {
		t.Fatal("sample attributes should round-trip")
	}
	if payload, ok := sample.Extensions().Core(hooks.SampleAttributes()); !ok {
		t.Fatal("expected sample extension payload")
	} else {
		payload.(map[string]any)["key"] = mutatedLiteral
		if sample.Attributes()["key"] != coreAttributeValue {
			t.Fatal("expected sample attribute clone to remain stable after extension mutation")
		}
	}

	permit := NewPermit(PermitData{
		Base:              BaseData{ID: "permit", CreatedAt: now},
		PermitNumber:      "PERMIT",
		Authority:         "Gov",
		ValidFrom:         now.Add(-time.Hour),
		ValidUntil:        now.Add(time.Hour),
		AllowedActivities: []string{"activity"},
		FacilityIDs:       []string{facility.ID()},
		ProtocolIDs:       []string{"protocol"},
		Notes:             strPtr("note"),
	})
	if permit.PermitNumber() == "" || permit.Authority() == "" || permit.Notes() == "" {
		t.Fatal("permit should expose fields")
	}
	if len(permit.AllowedActivities()) != 1 || len(permit.FacilityIDs()) != 1 || len(permit.ProtocolIDs()) != 1 {
		t.Fatal("permit should expose related ids")
	}
	if !permit.IsActive(now) || permit.IsExpired(now.Add(-2*time.Hour)) {
		t.Fatal("permit helpers should evaluate validity")
	}
	if permit.GetStatus(now).String() == "" {
		t.Fatal("permit contextual status should be available")
	}
	if len(permit.AllowedActivities()) != 1 || len(permit.FacilityIDs()) != 1 || len(permit.ProtocolIDs()) != 1 {
		t.Fatal("permit should expose related ids")
	}
	if permit.GetStatus(now).String() == "" {
		t.Fatal("permit contextual status should be available")
	}

	supply := NewSupplyItem(SupplyItemData{
		Base:           BaseData{ID: "supply", CreatedAt: now, UpdatedAt: now},
		SKU:            "SKU",
		Name:           "Feed",
		Description:    strPtr("desc"),
		QuantityOnHand: 1,
		Unit:           "kg",
		LotNumber:      strPtr("LOT"),
		FacilityIDs:    []string{facility.ID()},
		ProjectIDs:     []string{"proj"},
		ReorderLevel:   2,
		Extensions:     newCoreExtensionSet(hooks.SupplyItemAttributes(), map[string]any{"key": coreAttributeValue}),
		ExpiresAt:      &expires,
	})
	if supply.SKU() == "" || supply.Name() == "" || supply.Unit() == "" || supply.LotNumber() == "" {
		t.Fatal("supply should expose fields")
	}
	if len(supply.FacilityIDs()) != 1 || len(supply.ProjectIDs()) != 1 {
		t.Fatal("supply should expose related ids")
	}
	if !supply.RequiresReorder(now) || supply.Attributes()["key"] != coreAttributeValue {
		t.Fatal("supply helpers should report reorder and attributes")
	}
	if !supply.GetInventoryStatus(expires.Add(time.Hour)).IsExpired() {
		t.Fatal("supply should report expired after expiration")
	}
	if supply.Unit() == "" || supply.LotNumber() == "" {
		t.Fatal("supply should expose unit and lot number")
	}
	if len(supply.FacilityIDs()) != 1 || len(supply.ProjectIDs()) != 1 {
		t.Fatal("supply should expose related ids")
	}
	if qty := supply.QuantityOnHand(); qty != 1 {
		t.Fatalf("expected quantity 1, got %d", qty)
	}
	if payload, ok := supply.Extensions().Core(hooks.SupplyItemAttributes()); !ok {
		t.Fatal("expected supply extension payload")
	} else {
		payload.(map[string]any)["key"] = mutatedLiteral
		if supply.Attributes()["key"] != coreAttributeValue {
			t.Fatal("expected supply attribute clone to remain stable after extension mutation")
		}
	}

	// JSON round-trips cover MarshalJSON code paths for new facades
	for name, value := range map[string]any{
		"facility":    facility,
		"treatment":   treatment,
		"observation": observation,
		"sample":      sample,
		"permit":      permit,
		"supply":      supply,
	} {
		if _, err := json.Marshal(value); err != nil {
			t.Fatalf("marshal %s: %v", name, err)
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
	ext := organism.Extensions()
	if ext == nil {
		t.Fatalf("expected extensions set instance")
	}
	if ext.Hooks() != nil {
		t.Fatalf("expected no hooks for empty organism extensions")
	}
	if organism.ParentIDs() != nil {
		t.Fatalf("expected nil parent ids for empty organism")
	}

	breeding := NewBreedingUnit(BreedingUnitData{Base: BaseData{ID: "breed"}})
	if breeding.FemaleIDs() != nil || breeding.MaleIDs() != nil {
		t.Fatalf("expected female/male slices to be nil when empty")
	}

	procedure := NewProcedure(ProcedureData{Base: BaseData{ID: "proc"}, ProtocolID: "proto"})
	if _, ok := procedure.CohortID(); ok {
		t.Fatalf("expected cohort accessor on procedure to report missing value")
	}
	if _, ok := procedure.ProjectID(); ok {
		t.Fatalf("expected project accessor on procedure to report missing value")
	}
	if procedure.TreatmentIDs() != nil {
		t.Fatalf("expected nil treatment ids when unset")
	}
	if procedure.ObservationIDs() != nil {
		t.Fatalf("expected nil observation ids when unset")
	}
}

func TestDeepCloneAttributes(t *testing.T) {
	hooks := NewExtensionHookContext()
	nested := map[string]any{
		"level1": map[string]any{
			"level2": []any{map[string]any{"k": "v"}, []string{"a", "b"}},
		},
	}
	org := NewOrganism(OrganismData{Base: BaseData{ID: "deep"}, Extensions: newCoreExtensionSet(hooks.OrganismAttributes(), nested)})
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
