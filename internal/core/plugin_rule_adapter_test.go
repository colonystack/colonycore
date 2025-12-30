package core

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
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
	return domain.Organism{Organism: entitymodel.Organism{}}, false
}

func (v stubDomainView) FindHousingUnit(id string) (domain.HousingUnit, bool) {
	for _, housing := range v.housing {
		if housing.ID == id {
			return housing, true
		}
	}
	return domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{}}, false
}

func (v stubDomainView) FindFacility(id string) (domain.Facility, bool) {
	for _, facility := range v.facilities {
		if facility.ID == id {
			return facility, true
		}
	}
	return domain.Facility{Facility: entitymodel.Facility{}}, false
}

func (v stubDomainView) FindTreatment(id string) (domain.Treatment, bool) {
	for _, treatment := range v.treatments {
		if treatment.ID == id {
			return treatment, true
		}
	}
	return domain.Treatment{Treatment: entitymodel.Treatment{}}, false
}

func (v stubDomainView) FindObservation(id string) (domain.Observation, bool) {
	for _, observation := range v.observations {
		if observation.ID == id {
			return observation, true
		}
	}
	return domain.Observation{Observation: entitymodel.Observation{}}, false
}

func (v stubDomainView) FindSample(id string) (domain.Sample, bool) {
	for _, sample := range v.samples {
		if sample.ID == id {
			return sample, true
		}
	}
	return domain.Sample{Sample: entitymodel.Sample{}}, false
}

func (v stubDomainView) FindPermit(id string) (domain.Permit, bool) {
	for _, permit := range v.permits {
		if permit.ID == id {
			return permit, true
		}
	}
	return domain.Permit{Permit: entitymodel.Permit{}}, false
}

func (v stubDomainView) FindSupplyItem(id string) (domain.SupplyItem, bool) {
	for _, item := range v.supply {
		if item.ID == id {
			return item, true
		}
	}
	return domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{}}, false
}

func (v stubDomainView) FindProcedure(string) (domain.Procedure, bool) {
	return domain.Procedure{Procedure: entitymodel.Procedure{}}, false
}

func TestSampleViewAccessors(t *testing.T) {
	now := time.Date(2024, 5, 10, 12, 0, 0, 0, time.UTC)
	orgID := "org-1"
	note := "logged"
	testSampleColor := "red"
	sample := domain.Sample{Sample: entitymodel.Sample{ID: "sample-1",

		CreatedAt: now,

		UpdatedAt:       now,
		Identifier:      "S-1",
		SourceType:      "Organism",
		OrganismID:      &orgID,
		FacilityID:      "facility-1",
		CollectedAt:     now,
		Status:          domain.SampleStatusStored,
		StorageLocation: "Freezer A",
		AssayType:       "PCR",
		ChainOfCustody: []domain.SampleCustodyEvent{
			{
				Actor:     "tech",
				Location:  "lab",
				Timestamp: now,
				Notes:     &note,
			},
		}},
	}
	if err := sample.ApplySampleAttributes(map[string]any{"color": testSampleColor}); err != nil {
		t.Fatalf("apply sample attributes: %v", err)
	}

	view := newSampleView(sample)

	if view.ID() != sample.ID {
		t.Fatalf("expected ID %s, got %s", sample.ID, view.ID())
	}
	if view.Identifier() != sample.Identifier {
		t.Fatalf("expected identifier %s, got %s", sample.Identifier, view.Identifier())
	}
	if view.FacilityID() != sample.FacilityID {
		t.Fatalf("expected facility %s, got %s", sample.FacilityID, view.FacilityID())
	}
	if ident, ok := view.OrganismID(); !ok || ident != orgID {
		t.Fatalf("expected organism id %s, got %s (ok=%v)", orgID, ident, ok)
	}
	if _, ok := view.CohortID(); ok {
		t.Fatalf("expected cohort id to be missing")
	}
	if !view.CollectedAt().Equal(sample.CollectedAt) {
		t.Fatalf("expected collected at %v, got %v", sample.CollectedAt, view.CollectedAt())
	}
	if view.Status() != string(sample.Status) {
		t.Fatalf("expected status %s, got %s", sample.Status, view.Status())
	}
	if view.StorageLocation() != sample.StorageLocation {
		t.Fatalf("expected storage location %s, got %s", sample.StorageLocation, view.StorageLocation())
	}
	if view.AssayType() != sample.AssayType {
		t.Fatalf("expected assay type %s, got %s", sample.AssayType, view.AssayType())
	}

	custody := view.ChainOfCustody()
	if len(custody) != 1 || custody[0]["actor"] != "tech" {
		t.Fatalf("unexpected custody data: %+v", custody)
	}
	custody[0]["actor"] = testLiteralMutated
	if view.ChainOfCustody()[0]["actor"] != "tech" {
		t.Fatalf("expected custody clone to be immutable")
	}

	attrs := view.Attributes()
	if attrs["color"] != testSampleColor {
		t.Fatalf("expected attribute color %s, got %v", testSampleColor, attrs["color"])
	}
	attrs["color"] = "blue"
	if view.Attributes()["color"] != testSampleColor {
		t.Fatalf("expected attribute clone to remain %s", testSampleColor)
	}

	hookCtx := pluginapi.NewExtensionHookContext()
	corePayload, ok := view.Extensions().Core(hookCtx.SampleAttributes())
	if !ok {
		t.Fatalf("expected core sample attributes payload")
	}
	if corePayload.Map()["color"] != testSampleColor {
		t.Fatalf("expected extension payload color %s, got %v", testSampleColor, corePayload)
	}

	sourceCtx := pluginapi.NewSampleContext()
	if !view.GetSource().Equals(sourceCtx.Sources().Organism()) {
		t.Fatalf("expected sample source organism")
	}
	if !view.GetStatus().Equals(sourceCtx.Statuses().Stored()) {
		t.Fatalf("expected sample status stored")
	}
	if !view.IsAvailable() {
		t.Fatalf("expected stored sample to be available")
	}
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
		organisms:    []domain.Organism{{Organism: entitymodel.Organism{ID: organismID, HousingID: &housingID}}},
		housing:      []domain.HousingUnit{{HousingUnit: entitymodel.HousingUnit{ID: housingID}}},
		protocols:    []domain.Protocol{{Protocol: entitymodel.Protocol{ID: protocolID}}},
		facilities:   []domain.Facility{{Facility: entitymodel.Facility{ID: facilityID}}},
		treatments:   []domain.Treatment{{Treatment: entitymodel.Treatment{ID: treatmentID, ProcedureID: "proc"}}},
		observations: []domain.Observation{{Observation: entitymodel.Observation{ID: observationID}}},
		samples:      []domain.Sample{{Sample: entitymodel.Sample{ID: sampleID, FacilityID: facilityID}}},
		permits:      []domain.Permit{{Permit: entitymodel.Permit{ID: permitID}}},
		projects:     []domain.Project{{Project: entitymodel.Project{ID: "project-1"}}},
		supply:       []domain.SupplyItem{{SupplyItem: entitymodel.SupplyItem{ID: supplyID}}},
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

func TestToPluginChangesEncodesPayloads(t *testing.T) {
	changes := []domain.Change{{
		Entity: domain.EntityOrganism,
		Action: domain.ActionUpdate,
		Before: mustChangePayload(t, map[string]any{"id": "o1"}),
		After:  mustChangePayload(t, map[string]any{"id": "o2"}),
	}}
	converted := toPluginChanges(changes)
	if len(converted) != 1 {
		t.Fatalf("expected 1 converted change, got %d", len(converted))
	}
	before := converted[0].Before()
	if !before.Defined() {
		t.Fatalf("expected before payload to be defined")
	}
	var beforeData map[string]any
	if err := json.Unmarshal(before.Raw(), &beforeData); err != nil {
		t.Fatalf("unmarshal before payload: %v", err)
	}
	if beforeData["id"] != "o1" {
		t.Fatalf("expected before id 'o1', got %v", beforeData["id"])
	}
	after := converted[0].After()
	if !after.Defined() {
		t.Fatalf("expected after payload to be defined")
	}
	var afterData map[string]any
	if err := json.Unmarshal(after.Raw(), &afterData); err != nil {
		t.Fatalf("unmarshal after payload: %v", err)
	}
	if afterData["id"] != "o2" {
		t.Fatalf("expected after id 'o2', got %v", afterData["id"])
	}
}

func TestEncodeChangePayload(t *testing.T) {
	payload := encodeChangePayload(domain.UndefinedChangePayload())
	if payload.Defined() || payload.Raw() != nil {
		t.Fatalf("expected nil payload to be undefined")
	}

	payload = encodeChangePayload(domain.NewChangePayload(nil))
	if !payload.Defined() {
		t.Fatalf("expected empty payload to be defined")
	}
	if payload.Raw() != nil {
		t.Fatalf("expected empty payload to have nil raw bytes")
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
	domainFacility := domain.Facility{Facility: entitymodel.Facility{ID: "facility",

		CreatedAt: now,

		UpdatedAt:      now,
		Code:           "FAC-99",
		Name:           "Facility",
		Zone:           "Quarantine Zone",
		AccessPolicy:   "Restricted",
		HousingUnitIDs: []string{"H1"},
		ProjectIDs:     []string{"P1"}},
	}
	if err := domainFacility.ApplyEnvironmentBaselines(map[string]any{"temp": 21}); err != nil {
		t.Fatalf("apply facility baselines: %v", err)
	}
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
	facilityPayload := facility.CoreEnvironmentBaselinesPayload()
	if !facilityPayload.Defined() || facilityPayload.Map()["temp"] != 21 {
		t.Fatal("facility baseline payload should expose stored values")
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

	treatment := newTreatmentView(domain.Treatment{Treatment: entitymodel.Treatment{ID: "treatment",

		CreatedAt:         now,
		Name:              "Treatment",
		ProcedureID:       "proc",
		OrganismIDs:       []string{"org"},
		CohortIDs:         []string{"cohort"},
		DosagePlan:        "dose plan",
		AdministrationLog: []string{"dose"},
		AdverseEvents:     []string{"note"}},
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
	observationDomain := domain.Observation{Observation: entitymodel.Observation{ID: "observation",

		CreatedAt:   now,
		RecordedAt:  now,
		Observer:    "tech",
		ProcedureID: &procID,
		Notes:       strPtr("text")},
	}
	if err := observationDomain.ApplyObservationData(map[string]any{"score": 1}); err != nil {
		t.Fatalf("apply observation data: %v", err)
	}
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
	dataPayload := observation.CoreDataPayload()
	if !dataPayload.Defined() || dataPayload.Map()["score"] != 1 {
		t.Fatal("observation payload should expose structured data")
	}

	organID := "org"
	sampleDomain := domain.Sample{Sample: entitymodel.Sample{ID: "sample",

		CreatedAt:       now,
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
		}}},
	}
	if err := sampleDomain.ApplySampleAttributes(map[string]any{"k": "v"}); err != nil {
		t.Fatalf("ApplySampleAttributes: %v", err)
	}
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
	samplePayload := sample.CoreAttributesPayload()
	if !samplePayload.Defined() || samplePayload.Map()["k"] != "v" {
		t.Fatal("sample payload should expose stored attributes")
	}

	permit := newPermitView(domain.Permit{Permit: entitymodel.Permit{ID: "permit",

		CreatedAt:         now,
		PermitNumber:      "PERMIT",
		Authority:         "Gov",
		Status:            domain.PermitStatusApproved,
		ValidFrom:         now.Add(-time.Hour),
		ValidUntil:        now.Add(time.Hour),
		AllowedActivities: []string{"activity"},
		FacilityIDs:       []string{"facility"},
		ProtocolIDs:       []string{"protocol"},
		Notes:             strPtr("note")},
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

	supplyDomain := domain.SupplyItem{SupplyItem: entitymodel.SupplyItem{ID: "supply",

		CreatedAt: now,

		UpdatedAt:      now,
		SKU:            "SKU",
		Name:           "Feed",
		Description:    strPtr("desc"),
		QuantityOnHand: 1,
		Unit:           "kg",
		LotNumber:      strPtr("LOT"),
		FacilityIDs:    []string{"facility"},
		ProjectIDs:     []string{"project"},
		ReorderLevel:   2},
	}
	if err := supplyDomain.ApplySupplyAttributes(map[string]any{"k": "v"}); err != nil {
		t.Fatalf("ApplySupplyAttributes: %v", err)
	}
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
	supplyPayload := supply.CoreAttributesPayload()
	if !supplyPayload.Defined() || supplyPayload.Map()["k"] != "v" {
		t.Fatal("supply payload should expose stored attributes")
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

	domainOrg := domain.Organism{Organism: entitymodel.Organism{ID: "O1",

		CreatedAt: createdAt,

		UpdatedAt:  updatedAt,
		Name:       "Specimen",
		Species:    "Frogus",
		Line:       "LineA",
		LineID:     &lineID,
		StrainID:   &strainID,
		ParentIDs:  append([]string(nil), parentIDs...),
		Stage:      domain.StageAdult,
		HousingID:  &housingID,
		ProtocolID: &protocolID,
		ProjectID:  &projectID},
	}
	if err := domainOrg.SetCoreAttributes(attributes); err != nil {
		t.Fatalf("SetCoreAttributes: %v", err)
	}

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
	attrs["key"] = testLiteralMutated
	if refreshed := view.Attributes()["key"]; refreshed != "value" {
		t.Fatalf("expected attributes copy to remain unchanged, got %v", refreshed)
	}
	payload := view.CoreAttributesPayload()
	if !payload.Defined() {
		t.Fatalf("expected core attributes payload to be defined")
	}
	if payload.Map()["key"] != "value" {
		t.Fatalf("expected payload map to match stored attributes")
	}
	parentIDs[0] = "changed"
	if view.ParentIDs()[0] != "p1" {
		t.Fatalf("expected parent ids clone to remain stable")
	}
}

func TestHousingAndProtocolViews(t *testing.T) {
	createdAt := time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	domainUnit := domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "HU",

		CreatedAt: createdAt,

		UpdatedAt:   updatedAt,
		Name:        "Tank",
		FacilityID:  "North",
		Capacity:    12,
		Environment: domain.HousingEnvironmentHumid},
	}
	unitView := newHousingUnitView(domainUnit)
	if unitView.ID() != domainUnit.ID || unitView.Name() != domainUnit.Name {
		t.Fatalf("unexpected housing view %+v", unitView)
	}
	if got := unitView.Environment(); got != string(domainUnit.Environment) {
		t.Fatalf("unexpected housing environment: %s", got)
	}
	if unitView.FacilityID() != domainUnit.FacilityID || unitView.Capacity() != domainUnit.Capacity {
		t.Fatalf("unexpected housing facility/capacity: %s %d", unitView.FacilityID(), unitView.Capacity())
	}

	domainProtocol := domain.Protocol{Protocol: entitymodel.Protocol{ID: "PROTO",

		CreatedAt: createdAt,

		UpdatedAt:   updatedAt,
		Code:        "PR",
		Title:       "Protocol",
		Description: strPtr("Desc"),
		MaxSubjects: 5},
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
	domainOrganism := &domain.Organism{Organism: entitymodel.Organism{ID: "organism-1",

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
		Name:      "Test Organism",
		Species:   "TestSpecies",
		Line:      "TestLine",
		Stage:     "adult"},
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
	domainHousing := &domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "housing-1",

		CreatedAt: time.Now(),

		UpdatedAt:   time.Now(),
		Name:        "Test Tank",
		FacilityID:  "Lab A",
		Capacity:    10,
		Environment: "aquatic",
		State:       domain.HousingStateActive},
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

	t.Run("GetCurrentState returns contextual state reference", func(t *testing.T) {
		stateRef := housingView.GetCurrentState()
		if stateRef.String() != string(domainHousing.State) {
			t.Errorf("Expected state '%s', got '%s'", domainHousing.State, stateRef.String())
		}
		if !housingView.IsActiveState() {
			t.Error("Active housing should be treated as active")
		}
		if housingView.IsDecommissioned() {
			t.Error("Active housing should not be decommissioned")
		}
	})

	t.Run("GetCurrentState covers cleaning and decommissioned states", func(t *testing.T) {
		cleaning := newHousingUnitView(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "housing-2",
			Name:        "Cleaning Unit",
			FacilityID:  "Lab B",
			Capacity:    5,
			Environment: "terrestrial",
			State:       domain.HousingStateCleaning},
		})
		state := cleaning.GetCurrentState()
		if state.String() != string(domain.HousingStateCleaning) {
			t.Fatalf("expected cleaning state, got %s", state.String())
		}
		if cleaning.IsActiveState() {
			t.Fatal("cleaning housing should not be treated as active")
		}

		decommissioned := newHousingUnitView(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "housing-3",
			Name:        "Retired Unit",
			FacilityID:  "Lab C",
			Capacity:    2,
			Environment: "terrestrial",
			State:       domain.HousingStateDecommissioned},
		})
		if !decommissioned.IsDecommissioned() {
			t.Fatal("decommissioned housing should report terminal state")
		}
		if decommissioned.IsActiveState() {
			t.Fatal("decommissioned housing should not be active")
		}
	})

	t.Run("GetCurrentState handles quarantine and unknown states", func(t *testing.T) {
		quarantine := newHousingUnitView(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{ID: "housing-4",
			Name:        "Quarantine Unit",
			FacilityID:  "Lab D",
			Capacity:    2,
			Environment: "aquatic",
			State:       domain.HousingStateQuarantine},
		})
		if quarantine.GetCurrentState().String() != string(domain.HousingStateQuarantine) {
			t.Fatalf("expected quarantine state, got %s", quarantine.GetCurrentState().String())
		}
		if quarantine.IsDecommissioned() {
			t.Fatal("quarantine housing should not be decommissioned")
		}

		unknown := housingUnitView{
			baseView:    newBaseView("housing-5", time.Time{}, time.Time{}),
			name:        "Unknown State",
			facilityID:  "Lab E",
			capacity:    1,
			environment: "terrestrial",
			state:       "unexpected",
		}
		if !unknown.IsActiveState() {
			t.Fatal("unknown housing state should default to active reference")
		}
	})
}

func TestProtocolViewContextualAccessors(t *testing.T) {
	domainProtocol := &domain.Protocol{Protocol: entitymodel.Protocol{ID: "protocol-1",

		CreatedAt: time.Now(),

		UpdatedAt:   time.Now(),
		Code:        "P001",
		Title:       "Test Protocol",
		Description: strPtr("Test description"),
		MaxSubjects: 10,
		Status:      domain.ProtocolStatusApproved},
	}

	protocolView := newProtocolView(*domainProtocol)

	t.Run("GetCurrentStatus returns contextual status reference", func(t *testing.T) {
		statusRef := protocolView.GetCurrentStatus()
		if statusRef.String() != string(domain.ProtocolStatusApproved) {
			t.Errorf("Expected status '%s', got '%s'", domain.ProtocolStatusApproved, statusRef.String())
		}
	})

	t.Run("IsActiveProtocol returns correct active state", func(t *testing.T) {
		if !protocolView.IsActiveProtocol() {
			t.Error("Approved protocol should return true for IsActiveProtocol")
		}
	})

	t.Run("IsTerminalStatus returns correct terminal state", func(t *testing.T) {
		if protocolView.IsTerminalStatus() {
			t.Error("Approved protocol should return false for IsTerminalStatus")
		}
	})

	t.Run("CanAcceptNewSubjects returns correct capacity state", func(t *testing.T) {
		if !protocolView.CanAcceptNewSubjects() {
			t.Error("Approved protocol with capacity should accept new subjects")
		}
	})
}

func TestRuleViewFindOrganism(t *testing.T) {
	organisms := []domain.Organism{{Organism: entitymodel.Organism{ID: "org1",
		Name: "Organism 1"},
	}, {Organism: entitymodel.Organism{ID: "org2",
		Name: "Organism 2"},
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
	housings := []domain.HousingUnit{{HousingUnit: entitymodel.HousingUnit{ID: "house1",
		Name: "Housing 1"},
	}, {HousingUnit: entitymodel.HousingUnit{ID: "house2",
		Name: "Housing 2"},
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
