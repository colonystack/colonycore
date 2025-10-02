package frog

import (
	"context"
	"testing"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
)

// fakeRegistry captures registrations to exercise Plugin.Register.
type fakeRegistry struct {
	schemas  map[string]map[string]any
	rules    []pluginapi.Rule
	datasets int
}

func (r *fakeRegistry) RegisterSchema(entity string, schema map[string]any) {
	if r.schemas == nil {
		r.schemas = make(map[string]map[string]any)
	}
	r.schemas[entity] = schema
}
func (r *fakeRegistry) RegisterRule(rule pluginapi.Rule)                    { r.rules = append(r.rules, rule) }
func (r *fakeRegistry) RegisterDatasetTemplate(_ datasetapi.Template) error { r.datasets++; return nil }

// fakeView implements pluginapi.RuleView (alias of domain.RuleView).
type fakeView struct {
	organisms []domain.Organism
	housing   []domain.HousingUnit
}

func (v fakeView) ListOrganisms() []domain.Organism       { return v.organisms }
func (v fakeView) ListHousingUnits() []domain.HousingUnit { return v.housing }
func (v fakeView) FindHousingUnit(id string) (domain.HousingUnit, bool) {
	for _, h := range v.housing {
		if h.ID == id {
			return h, true
		}
	}
	return domain.HousingUnit{}, false
}
func (v fakeView) ListProtocols() []domain.Protocol { return nil }
func (v fakeView) FindOrganism(id string) (domain.Organism, bool) {
	for _, o := range v.organisms {
		if o.ID == id {
			return o, true
		}
	}
	return domain.Organism{}, false
}

// TestFrogPluginRegisterAndRuleEvaluation covers plugin registration and rule violation generation.
func TestFrogPluginRegisterAndRuleEvaluation(t *testing.T) {
	var reg fakeRegistry
	if err := New().Register(&reg); err != nil {
		t.Fatalf("register: %v", err)
	}
	if len(reg.schemas) == 0 || len(reg.rules) == 0 || reg.datasets == 0 {
		t.Fatalf("expected registrations captured: %+v", reg)
	}
	// locate frog habitat rule
	var habitat domain.Rule
	for _, r := range reg.rules {
		if r.Name() == frogHabitatRuleName {
			habitat = r
			break
		}
	}
	if habitat == nil {
		t.Fatalf("frog habitat rule not registered")
	}
	// Evaluate with one frog in non-humid housing to trigger warning
	housingID := "H1"
	view := fakeView{
		organisms: []domain.Organism{{Base: domain.Base{ID: "O1"}, Species: "FrogX", HousingID: &housingID}},
		housing:   []domain.HousingUnit{{Base: domain.Base{ID: housingID}, Environment: "dry"}},
	}
	res, err := habitat.Evaluate(context.Background(), view, nil)
	if err != nil || len(res.Violations) != 1 {
		t.Fatalf("expected 1 violation: %+v err=%v", res, err)
	}
}
