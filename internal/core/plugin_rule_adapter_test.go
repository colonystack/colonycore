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
	organisms []domain.Organism
	housing   []domain.HousingUnit
	protocols []domain.Protocol
}

func (v stubDomainView) ListOrganisms() []domain.Organism       { return v.organisms }
func (v stubDomainView) ListHousingUnits() []domain.HousingUnit { return v.housing }
func (v stubDomainView) ListProtocols() []domain.Protocol       { return v.protocols }

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

func TestAdaptPluginRuleBridgesDomainInterfaces(t *testing.T) {
	housingID := "housing-1"
	organismID := "organism-1"
	protocolID := "protocol-1"
	view := stubDomainView{
		organisms: []domain.Organism{{Base: domain.Base{ID: organismID}, HousingID: &housingID}},
		housing:   []domain.HousingUnit{{Base: domain.Base{ID: housingID}}},
		protocols: []domain.Protocol{{Base: domain.Base{ID: protocolID}}},
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
	attributes := map[string]any{"key": "value"}

	domainOrg := domain.Organism{
		Base:       domain.Base{ID: "O1", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Name:       "Specimen",
		Species:    "Frogus",
		Line:       "LineA",
		Stage:      domain.StageAdult,
		HousingID:  &housingID,
		ProtocolID: &protocolID,
		ProjectID:  &projectID,
		Attributes: attributes,
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
}

func TestHousingAndProtocolViews(t *testing.T) {
	createdAt := time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	domainUnit := domain.HousingUnit{
		Base:        domain.Base{ID: "HU", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Name:        "Tank",
		Facility:    "North",
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
	if unitView.Facility() != domainUnit.Facility || unitView.Capacity() != domainUnit.Capacity {
		t.Fatalf("unexpected housing facility/capacity: %s %d", unitView.Facility(), unitView.Capacity())
	}

	domainProtocol := domain.Protocol{
		Base:        domain.Base{ID: "PROTO", CreatedAt: createdAt, UpdatedAt: updatedAt},
		Code:        "PR",
		Title:       "Protocol",
		Description: "Desc",
		MaxSubjects: 5,
	}
	protocolView := newProtocolView(domainProtocol)
	if protocolView.ID() != domainProtocol.ID || protocolView.Code() != domainProtocol.Code {
		t.Fatalf("unexpected protocol view %+v", protocolView)
	}
	if protocolView.Title() != domainProtocol.Title {
		t.Fatalf("unexpected protocol title: %s", protocolView.Title())
	}
	if protocolView.Description() != domainProtocol.Description || protocolView.MaxSubjects() != domainProtocol.MaxSubjects {
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
