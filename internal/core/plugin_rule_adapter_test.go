package core

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
)

type capturingRule struct {
	seenOrganism     string
	seenHousing      string
	seenHousingCount int
	seenProtocols    int
	seenFacilities   int
	seenTreatments   int
	seenObservations int
	seenSamples      int
	seenPermits      int
	seenProjects     int
	seenSupplyItems  int
	seenChanges      int
}

func (r *capturingRule) Name() string { return "capture" }

func (r *capturingRule) Evaluate(_ context.Context, view pluginapi.RuleView, changes []pluginapi.Change) (pluginapi.Result, error) {
	if view != nil {
		organisms := view.ListOrganisms()
		if len(organisms) > 0 {
			r.seenOrganism = organisms[0].ID()
		}
		housingUnits := view.ListHousingUnits()
		r.seenHousingCount = len(housingUnits)
		if housing, ok := view.FindHousingUnit("housing-1"); ok {
			r.seenHousing = housing.ID()
		}
		r.seenProtocols = len(view.ListProtocols())
		r.seenFacilities = len(view.ListFacilities())
		r.seenTreatments = len(view.ListTreatments())
		r.seenObservations = len(view.ListObservations())
		r.seenSamples = len(view.ListSamples())
		r.seenPermits = len(view.ListPermits())
		r.seenProjects = len(view.ListProjects())
		r.seenSupplyItems = len(view.ListSupplyItems())
		view.FindFacility("facility-1")
		view.FindTreatment("treatment-1")
		view.FindObservation("observation-1")
		view.FindSample("sample-1")
		view.FindPermit("permit-1")
		view.FindSupplyItem("supply-1")
	}
	r.seenChanges = len(changes)
	entities := pluginapi.NewEntityContext()

	violation, err := pluginapi.NewViolationBuilder().
		WithRule(r.Name()).
		WithEntity(entities.Organism()).
		BuildWarning()
	if err != nil {
		return pluginapi.Result{}, err
	}

	return pluginapi.NewResultBuilder().
		AddViolation(violation).
		Build(), nil
}

type stubDomainView struct {
	organisms    []domain.Organism
	housing      []domain.HousingUnit
	protocols    []domain.Protocol
	facilities   []domain.Facility
	treatments   []domain.Treatment
	observations []domain.Observation
	samples      []domain.Sample
	permits      []domain.Permit
	projects     []domain.Project
	supply       []domain.SupplyItem
}

func (v stubDomainView) ListOrganisms() []domain.Organism       { return v.organisms }
func (v stubDomainView) ListHousingUnits() []domain.HousingUnit { return v.housing }
func (v stubDomainView) ListProtocols() []domain.Protocol       { return v.protocols }
func (v stubDomainView) ListFacilities() []domain.Facility      { return v.facilities }
func (v stubDomainView) ListTreatments() []domain.Treatment     { return v.treatments }
func (v stubDomainView) ListObservations() []domain.Observation { return v.observations }
func (v stubDomainView) ListSamples() []domain.Sample           { return v.samples }
func (v stubDomainView) ListPermits() []domain.Permit           { return v.permits }
func (v stubDomainView) ListProjects() []domain.Project         { return v.projects }
func (v stubDomainView) ListSupplyItems() []domain.SupplyItem   { return v.supply }

func (v stubDomainView) FindOrganism(id string) (domain.Organism, bool) {
	for _, organism := range v.organisms {
		if organism.ID == id {
			return organism, true
		}
	}
	return domain.Organism{}, false
}

func (v stubDomainView) FindHousingUnit(id string) (domain.HousingUnit, bool) {
	for _, housing := range v.housing {
		if housing.ID == id {
			return housing, true
		}
	}
	return domain.HousingUnit{}, false
}

func (v stubDomainView) FindFacility(id string) (domain.Facility, bool) {
	for _, facility := range v.facilities {
		if facility.ID == id {
			return facility, true
		}
	}
	return domain.Facility{}, false
}

func (v stubDomainView) FindTreatment(id string) (domain.Treatment, bool) {
	for _, treatment := range v.treatments {
		if treatment.ID == id {
			return treatment, true
		}
	}
	return domain.Treatment{}, false
}

func (v stubDomainView) FindObservation(id string) (domain.Observation, bool) {
	for _, observation := range v.observations {
		if observation.ID == id {
			return observation, true
		}
	}
	return domain.Observation{}, false
}

func (v stubDomainView) FindSample(id string) (domain.Sample, bool) {
	for _, sample := range v.samples {
		if sample.ID == id {
			return sample, true
		}
	}
	return domain.Sample{}, false
}

func (v stubDomainView) FindPermit(id string) (domain.Permit, bool) {
	for _, permit := range v.permits {
		if permit.ID == id {
			return permit, true
		}
	}
	return domain.Permit{}, false
}

func (v stubDomainView) FindSupplyItem(id string) (domain.SupplyItem, bool) {
	for _, item := range v.supply {
		if item.ID == id {
			return item, true
		}
	}
	return domain.SupplyItem{}, false
}

func TestAdaptPluginRuleBridgesDomainInterfaces(t *testing.T) {
	housingID := "housing-1"
	organismID := "organism-1"
	protocolID := "protocol-1"
	facilityID := "facility-1"
	treatmentID := "treatment-1"
	observationID := "observation-1"
	sampleID := "sample-1"
	permitID := "permit-1"
	supplyID := "supply-1"
	view := stubDomainView{
		organisms:    []domain.Organism{{Base: domain.Base{ID: organismID}, HousingID: &housingID}},
		housing:      []domain.HousingUnit{{Base: domain.Base{ID: housingID}}},
		protocols:    []domain.Protocol{{Base: domain.Base{ID: protocolID}}},
		facilities:   []domain.Facility{{Base: domain.Base{ID: facilityID}}},
		treatments:   []domain.Treatment{{Base: domain.Base{ID: treatmentID}, ProcedureID: "proc"}},
		observations: []domain.Observation{{Base: domain.Base{ID: observationID}}},
		samples:      []domain.Sample{{Base: domain.Base{ID: sampleID}, FacilityID: facilityID}},
		permits:      []domain.Permit{{Base: domain.Base{ID: permitID}}},
		projects:     []domain.Project{{Base: domain.Base{ID: "project-1"}}},
		supply:       []domain.SupplyItem{{Base: domain.Base{ID: supplyID}}},
	}
	rule := &capturingRule{}
	adapted := adaptPluginRule(rule)
	if adapted == nil {
		t.Fatalf("expected adapted rule")
	}
	if adapted.Name() != rule.Name() {
		t.Fatalf("expected adapted rule to expose plugin rule name")
	}
	changes := []domain.Change{{Entity: domain.EntityOrganism}}
	result, err := adapted.Evaluate(context.Background(), view, changes)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(result.Violations) != 1 || result.Violations[0].Rule != rule.Name() {
		t.Fatalf("expected violation from plugin rule, got %+v", result)
	}
	if rule.seenOrganism != organismID {
		t.Fatalf("expected plugin rule to observe organism %s, got %s", organismID, rule.seenOrganism)
	}
	if rule.seenHousing != housingID {
		t.Fatalf("expected plugin rule to observe housing %s, got %s", housingID, rule.seenHousing)
	}
	if rule.seenHousingCount != len(view.housing) {
		t.Fatalf("expected plugin rule to observe %d housing units, got %d", len(view.housing), rule.seenHousingCount)
	}
	if rule.seenProtocols != len(view.protocols) {
		t.Fatalf("expected plugin rule to observe %d protocols, got %d", len(view.protocols), rule.seenProtocols)
	}
	if rule.seenFacilities != len(view.facilities) {
		t.Fatalf("expected plugin rule to observe %d facilities, got %d", len(view.facilities), rule.seenFacilities)
	}
	if rule.seenTreatments != len(view.treatments) {
		t.Fatalf("expected plugin rule to observe %d treatments, got %d", len(view.treatments), rule.seenTreatments)
	}
	if rule.seenObservations != len(view.observations) {
		t.Fatalf("expected plugin rule to observe %d observations, got %d", len(view.observations), rule.seenObservations)
	}
	if rule.seenSamples != len(view.samples) {
		t.Fatalf("expected plugin rule to observe %d samples, got %d", len(view.samples), rule.seenSamples)
	}
	if rule.seenPermits != len(view.permits) {
		t.Fatalf("expected plugin rule to observe %d permits, got %d", len(view.permits), rule.seenPermits)
	}
	if rule.seenProjects != len(view.projects) {
		t.Fatalf("expected plugin rule to observe %d projects, got %d", len(view.projects), rule.seenProjects)
	}
	if rule.seenSupplyItems != len(view.supply) {
		t.Fatalf("expected plugin rule to observe %d supply items, got %d", len(view.supply), rule.seenSupplyItems)
	}
	if rule.seenChanges != len(changes) {
		t.Fatalf("expected plugin rule to observe %d changes, got %d", len(changes), rule.seenChanges)
	}
}

type nilViewRule struct {
	gotNil bool
}

func (r *nilViewRule) Name() string { return "nil" }

func (r *nilViewRule) Evaluate(_ context.Context, view pluginapi.RuleView, _ []pluginapi.Change) (pluginapi.Result, error) {
	r.gotNil = view == nil
	return pluginapi.Result{}, nil
}

func TestNewViewAccessors(t *testing.T) {
	now := time.Now()
	domainFacility := domain.Facility{
		Base:           domain.Base{ID: "facility", CreatedAt: now, UpdatedAt: now},
		Code:           "FAC-99",
		Name:           "Facility",
		Zone:           "Quarantine Zone",
		AccessPolicy:   "Restricted",
		HousingUnitIDs: []string{"H1"},
		ProjectIDs:     []string{"P1"},
	}
	domainFacility.SetEnvironmentBaselines(map[string]any{"temp": 21})
	facility := newFacilityView(domainFacility)
	if facility.Name() == "" || facility.Zone() == "" || facility.AccessPolicy() == "" {
		t.Fatal("facility view should expose base fields")
	}
	if len(facility.HousingUnitIDs()) != 1 || len(facility.ProjectIDs()) != 1 {
		t.Fatal("facility view should expose related ids")
	}
	if facility.EnvironmentBaselines()["temp"] != 21 {
		t.Fatal("facility baselines should round-trip")
	}
	if facility.Code() != "FAC-99" {
		t.Fatalf("expected facility code to round-trip, got %q", facility.Code())
	}
	if !facility.GetZone().IsQuarantine() {
		t.Fatal("facility zone contextual accessor should report quarantine")
	}
	if !facility.GetAccessPolicy().IsRestricted() {
		t.Fatal("facility access policy should report restricted")
	}

	treatment := newTreatmentView(domain.Treatment{
		Base:              domain.Base{ID: "treatment", CreatedAt: now},
		Name:              "Treatment",
		ProcedureID:       "proc",
		OrganismIDs:       []string{"org"},
		CohortIDs:         []string{"cohort"},
		DosagePlan:        "dose plan",
		AdministrationLog: []string{"dose"},
		AdverseEvents:     []string{"note"},
	})
	if treatment.Name() == "" || treatment.ProcedureID() == "" {
		t.Fatal("treatment view should expose base fields")
	}
	if len(treatment.OrganismIDs()) != 1 || len(treatment.CohortIDs()) != 1 {
		t.Fatal("treatment view should expose related ids")
	}
	if treatment.DosagePlan() == "" {
		t.Fatal("treatment should expose dosage plan")
	}
	if !treatment.HasAdverseEvents() || !treatment.IsCompleted() {
		t.Fatal("treatment view helpers should reflect log state")
	}

	procID := "proc"
	observationDomain := domain.Observation{
		Base:        domain.Base{ID: "observation", CreatedAt: now},
		RecordedAt:  now,
		Observer:    "tech",
		ProcedureID: &procID,
		Notes:       strPtr("text"),
	}
	observationDomain.SetData(map[string]any{"score": 1})
	observation := newObservationView(observationDomain)
	if observation.Observer() == "" || observation.Notes() == "" {
		t.Fatal("observation view should expose observer")
	}
	if _, ok := observation.ProcedureID(); !ok {
		t.Fatal("observation should expose procedure id when set")
	}
	if !observation.GetDataShape().HasNarrativeNotes() {
		t.Fatal("observation data shape should report narrative notes")
	}

	organID := "org"
	sampleDomain := domain.Sample{
		Base:            domain.Base{ID: "sample", CreatedAt: now},
		Identifier:      "S1",
		SourceType:      "organism",
		OrganismID:      &organID,
		FacilityID:      "facility",
		CollectedAt:     now,
		Status:          domain.SampleStatusStored,
		StorageLocation: "freezer",
		AssayType:       "assay",
		ChainOfCustody: []domain.SampleCustodyEvent{{
			Actor:     "tech",
			Location:  "lab",
			Timestamp: now,
			Notes:     strPtr("note"),
		}},
	}
	sampleDomain.SetAttributes(map[string]any{"k": "v"})
	sample := newSampleView(sampleDomain)
	if sample.Identifier() == "" || sample.AssayType() == "" || sample.StorageLocation() == "" {
		t.Fatal("sample view should expose base fields")
	}
	if _, ok := sample.OrganismID(); !ok {
		t.Fatal("sample should expose organism id when set")
	}
	if len(sample.ChainOfCustody()) != 1 {
		t.Fatal("sample view should expose custody events")
	}
	if !sample.GetStatus().IsAvailable() || !sample.GetSource().IsOrganismDerived() {
		t.Fatal("sample status contextual helper should report availability")
	}

	permit := newPermitView(domain.Permit{
		Base:              domain.Base{ID: "permit", CreatedAt: now},
		PermitNumber:      "PERMIT",
		Authority:         "Gov",
		Status:            domain.PermitStatusActive,
		ValidFrom:         now.Add(-time.Hour),
		ValidUntil:        now.Add(time.Hour),
		AllowedActivities: []string{"activity"},
		FacilityIDs:       []string{"facility"},
		ProtocolIDs:       []string{"protocol"},
		Notes:             strPtr("note"),
	})
	if permit.PermitNumber() == "" || permit.Authority() == "" || permit.Notes() == "" {
		t.Fatal("permit view should expose base fields")
	}
	if len(permit.AllowedActivities()) != 1 || len(permit.FacilityIDs()) != 1 || len(permit.ProtocolIDs()) != 1 {
		t.Fatal("permit view should expose related ids")
	}
	if !permit.IsActive(now) || permit.IsExpired(now.Add(-2*time.Hour)) {
		t.Fatal("permit view should consider validity window active")
	}

	supplyDomain := domain.SupplyItem{
		Base:           domain.Base{ID: "supply", CreatedAt: now, UpdatedAt: now},
		SKU:            "SKU",
		Name:           "Feed",
		Description:    strPtr("desc"),
		QuantityOnHand: 1,
		Unit:           "kg",
		LotNumber:      strPtr("LOT"),
		FacilityIDs:    []string{"facility"},
		ProjectIDs:     []string{"project"},
		ReorderLevel:   2,
	}
	supplyDomain.SetAttributes(map[string]any{"k": "v"})
	supply := newSupplyItemView(supplyDomain)
	if supply.SKU() == "" || supply.Name() == "" || supply.Description() == "" || supply.Unit() == "" || supply.LotNumber() == "" {
		t.Fatal("supply view should expose base fields")
	}
	if len(supply.FacilityIDs()) != 1 || len(supply.ProjectIDs()) != 1 {
		t.Fatal("supply view should expose related ids")
	}
	if !supply.RequiresReorder(now) {
		t.Fatal("supply view should report reorder when quantity below threshold")
	}
	if supply.Attributes()["k"] != "v" {
		t.Fatal("supply attributes should round-trip")
	}
}

func TestAdaptPluginRuleHandlesNilInputs(t *testing.T) {
	if adaptPluginRule(nil) != nil {
		t.Fatalf("expected nil adapt result for nil rule")
	}
	rule := &nilViewRule{}
	adapted := adaptPluginRule(rule)
	if adapted == nil {
		t.Fatalf("expected adapter to wrap rule")
	}
	if _, err := adapted.Evaluate(context.Background(), nil, nil); err != nil {
		t.Fatalf("evaluate with nil inputs: %v", err)
	}
	if !rule.gotNil {
		t.Fatalf("expected plugin rule to receive nil view")
	}
}

func TestOrganismViewAccessors(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	housingID := "H1"
	protocolID := "P1"
	projectID := "PRJ"
	lineID := "line-id"
	strainID := "strain-id"
	parentIDs := []string{"p1", "p2"}
	attributes := map[string]any{"key": "value"}

	domainOrg := domain.Organism{
		Base:       domain.Base{ID: "O1", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Name:       "Specimen",
		Species:    "Frogus",
		Line:       "LineA",
		LineID:     &lineID,
		StrainID:   &strainID,
		ParentIDs:  append([]string(nil), parentIDs...),
		Stage:      domain.StageAdult,
		HousingID:  &housingID,
		ProtocolID: &protocolID,
		ProjectID:  &projectID,
	}
	domainOrg.SetAttributes(attributes)

	view := newOrganismView(domainOrg)

	if view.ID() != domainOrg.ID {
		t.Fatalf("unexpected id: %s", view.ID())
	}
	if !view.CreatedAt().Equal(createdAt) || !view.UpdatedAt().Equal(updatedAt) {
		t.Fatalf("unexpected timestamps: %v %v", view.CreatedAt(), view.UpdatedAt())
	}
	if view.Name() != domainOrg.Name || view.Species() != domainOrg.Species {
		t.Fatalf("unexpected name/species: %s %s", view.Name(), view.Species())
	}
	if view.Line() != domainOrg.Line {
		t.Fatalf("unexpected line: %s", view.Line())
	}
	if got, ok := view.LineID(); !ok || got != lineID {
		t.Fatalf("unexpected line id: %s (%v)", got, ok)
	}
	if got, ok := view.StrainID(); !ok || got != strainID {
		t.Fatalf("unexpected strain id: %s (%v)", got, ok)
	}
	if parents := view.ParentIDs(); len(parents) != len(parentIDs) || parents[0] != "p1" {
		t.Fatalf("unexpected parent ids: %+v", parents)
	}
	if view.Stage() != pluginapi.LifecycleStage(domain.StageAdult) {
		t.Fatalf("unexpected stage: %s", view.Stage())
	}
	if _, ok := view.CohortID(); ok {
		t.Fatalf("expected no cohort id")
	}
	if got, ok := view.HousingID(); !ok || got != housingID {
		t.Fatalf("unexpected housing id: %q %v", got, ok)
	}
	if got, ok := view.ProtocolID(); !ok || got != protocolID {
		t.Fatalf("unexpected protocol id: %q %v", got, ok)
	}
	if got, ok := view.ProjectID(); !ok || got != projectID {
		t.Fatalf("unexpected project id: %q %v", got, ok)
	}

	attrs := view.Attributes()
	attrs["key"] = "mutated"
	if refreshed := view.Attributes()["key"]; refreshed != "value" {
		t.Fatalf("expected attributes copy to remain unchanged, got %v", refreshed)
	}
	parentIDs[0] = "changed"
	if view.ParentIDs()[0] != "p1" {
		t.Fatalf("expected parent ids clone to remain stable")
	}
}

func TestHousingAndProtocolViews(t *testing.T) {
	createdAt := time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	domainUnit := domain.HousingUnit{
		Base:        domain.Base{ID: "HU", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Name:        "Tank",
		FacilityID:  "North",
		Capacity:    12,
		Environment: "humid",
	}
	unitView := newHousingUnitView(domainUnit)
	if unitView.ID() != domainUnit.ID || unitView.Name() != domainUnit.Name {
		t.Fatalf("unexpected housing view %+v", unitView)
	}
	if got := unitView.Environment(); got != domainUnit.Environment {
		t.Fatalf("unexpected housing environment: %s", got)
	}
	if unitView.FacilityID() != domainUnit.FacilityID || unitView.Capacity() != domainUnit.Capacity {
		t.Fatalf("unexpected housing facility/capacity: %s %d", unitView.FacilityID(), unitView.Capacity())
	}

	domainProtocol := domain.Protocol{
		Base:        domain.Base{ID: "PROTO", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Code:        "PR",
		Title:       "Protocol",
		Description: strPtr("Desc"),
		MaxSubjects: 5,
	}
	protocolView := newProtocolView(domainProtocol)
	if protocolView.ID() != domainProtocol.ID || protocolView.Code() != domainProtocol.Code {
		t.Fatalf("unexpected protocol view %+v", protocolView)
	}
	if protocolView.Title() != domainProtocol.Title {
		t.Fatalf("unexpected protocol title: %s", protocolView.Title())
	}
	if protocolView.Description() != *domainProtocol.Description || protocolView.MaxSubjects() != domainProtocol.MaxSubjects {
		t.Fatalf("unexpected protocol details")
	}
}

func TestEmptyViewHelpers(t *testing.T) {
	if views := newOrganismViews(nil); views != nil {
		t.Fatalf("expected nil organism views slice")
	}
	if views := newHousingUnitViews(nil); views != nil {
		t.Fatalf("expected nil housing views slice")
	}
	if views := newProtocolViews(nil); views != nil {
		t.Fatalf("expected nil protocol views slice")
	}
}

func TestOrganismViewContextualAccessors(t *testing.T) {
	domainOrganism := &domain.Organism{
		Base: domain.Base{
			ID:        "organism-1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:    "Test Organism",
		Species: "TestSpecies",
		Line:    "TestLine",
		Stage:   "adult",
	}

	organismView := newOrganismView(*domainOrganism)

	t.Run("GetCurrentStage returns contextual stage reference", func(t *testing.T) {
		stageRef := organismView.GetCurrentStage()
		if stageRef.String() != "adult" {
			t.Errorf("Expected stage 'adult', got '%s'", stageRef.String())
		}
	})

	t.Run("IsActive returns correct lifecycle state", func(t *testing.T) {
		if !organismView.IsActive() {
			t.Error("Adult organism should be active")
		}
	})

	t.Run("IsRetired returns correct retirement state", func(t *testing.T) {
		if organismView.IsRetired() {
			t.Error("Adult organism should not be retired")
		}
	})

	t.Run("IsDeceased returns correct death state", func(t *testing.T) {
		if organismView.IsDeceased() {
			t.Error("Adult organism should not be deceased")
		}
	})
}

func TestHousingUnitViewContextualAccessors(t *testing.T) {
	domainHousing := &domain.HousingUnit{
		Base: domain.Base{
			ID:        "housing-1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:        "Test Tank",
		FacilityID:  "Lab A",
		Capacity:    10,
		Environment: "aquatic",
	}

	housingView := newHousingUnitView(*domainHousing)

	t.Run("GetEnvironmentType returns contextual environment reference", func(t *testing.T) {
		envRef := housingView.GetEnvironmentType()
		if envRef.String() != "aquatic" {
			t.Errorf("Expected environment 'aquatic', got '%s'", envRef.String())
		}
	})

	t.Run("IsAquaticEnvironment returns correct aquatic state", func(t *testing.T) {
		if !housingView.IsAquaticEnvironment() {
			t.Error("Aquatic housing should return true for IsAquaticEnvironment")
		}
	})

	t.Run("IsHumidEnvironment returns correct humidity state", func(t *testing.T) {
		// Aquatic environments are also considered humid
		if !housingView.IsHumidEnvironment() {
			t.Error("Aquatic housing should return true for IsHumidEnvironment")
		}
	})

	t.Run("SupportsSpecies returns correct species compatibility", func(t *testing.T) {
		if !housingView.SupportsSpecies("fish") {
			t.Error("Aquatic housing should support fish")
		}
		if housingView.SupportsSpecies("bird") {
			t.Error("Aquatic housing should not support bird")
		}
	})
}

func TestProtocolViewContextualAccessors(t *testing.T) {
	domainProtocol := &domain.Protocol{
		Base: domain.Base{
			ID:        "protocol-1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Code:        "P001",
		Title:       "Test Protocol",
		Description: strPtr("Test description"),
		MaxSubjects: 10,
		Status:      "active",
	}

	protocolView := newProtocolView(*domainProtocol)

	t.Run("GetCurrentStatus returns contextual status reference", func(t *testing.T) {
		statusRef := protocolView.GetCurrentStatus()
		if statusRef.String() != "active" {
			t.Errorf("Expected status 'active', got '%s'", statusRef.String())
		}
	})

	t.Run("IsActiveProtocol returns correct active state", func(t *testing.T) {
		if !protocolView.IsActiveProtocol() {
			t.Error("Active protocol should return true for IsActiveProtocol")
		}
	})

	t.Run("IsTerminalStatus returns correct terminal state", func(t *testing.T) {
		if protocolView.IsTerminalStatus() {
			t.Error("Active protocol should return false for IsTerminalStatus")
		}
	})

	t.Run("CanAcceptNewSubjects returns correct capacity state", func(t *testing.T) {
		if !protocolView.CanAcceptNewSubjects() {
			t.Error("Active protocol with capacity should accept new subjects")
		}
	})
}

func TestRuleViewFindOrganism(t *testing.T) {
	organisms := []domain.Organism{
		{
			Base: domain.Base{ID: "org1"},
			Name: "Organism 1",
		},
		{
			Base: domain.Base{ID: "org2"},
			Name: "Organism 2",
		},
	}

	domainView := stubDomainView{organisms: organisms}
	ruleView := adaptRuleView(domainView)

	t.Run("FindOrganism returns correct organism when found", func(t *testing.T) {
		organism, found := ruleView.FindOrganism("org1")
		if !found {
			t.Error("Should find organism with ID 'org1'")
		}
		if organism.ID() != "org1" {
			t.Errorf("Expected organism ID 'org1', got '%s'", organism.ID())
		}
	})

	t.Run("FindOrganism returns false when not found", func(t *testing.T) {
		_, found := ruleView.FindOrganism("nonexistent")
		if found {
			t.Error("Should not find organism with nonexistent ID")
		}
	})
}

func TestRuleViewFindHousing(t *testing.T) {
	housings := []domain.HousingUnit{
		{
			Base: domain.Base{ID: "house1"},
			Name: "Housing 1",
		},
		{
			Base: domain.Base{ID: "house2"},
			Name: "Housing 2",
		},
	}

	domainView := stubDomainView{housing: housings}
	ruleView := adaptRuleView(domainView)

	t.Run("FindHousingUnit returns correct housing when found", func(t *testing.T) {
		housing, found := ruleView.FindHousingUnit("house1")
		if !found {
			t.Error("Should find housing with ID 'house1'")
		}
		if housing.ID() != "house1" {
			t.Errorf("Expected housing ID 'house1', got '%s'", housing.ID())
		}
	})

	t.Run("FindHousingUnit returns false when not found", func(t *testing.T) {
		_, found := ruleView.FindHousingUnit("nonexistent")
		if found {
			t.Error("Should not find housing with nonexistent ID")
		}
	})
}
