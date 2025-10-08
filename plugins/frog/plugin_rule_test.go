package frog

import (
	"context"
	"testing"
	"time"

	"colonycore/pkg/datasetapi"
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

// fakeView implements pluginapi.RuleView for exercising rule evaluation paths.
type fakeView struct {
	organisms []pluginapi.OrganismView
	housing   []pluginapi.HousingUnitView
}

func (v fakeView) ListOrganisms() []pluginapi.OrganismView       { return v.organisms }
func (v fakeView) ListHousingUnits() []pluginapi.HousingUnitView { return v.housing }
func (v fakeView) ListProtocols() []pluginapi.ProtocolView       { return nil }

func (v fakeView) FindHousingUnit(id string) (pluginapi.HousingUnitView, bool) {
	for _, h := range v.housing {
		if h.ID() == id {
			return h, true
		}
	}
	return nil, false
}

func (v fakeView) FindOrganism(id string) (pluginapi.OrganismView, bool) {
	for _, o := range v.organisms {
		if o.ID() == id {
			return o, true
		}
	}
	return nil, false
}

type stubOrganism struct {
	id        string
	species   string
	housingID *string
}

func (o stubOrganism) ID() string                    { return o.id }
func (stubOrganism) CreatedAt() time.Time            { return time.Time{} }
func (stubOrganism) UpdatedAt() time.Time            { return time.Time{} }
func (stubOrganism) Name() string                    { return "" }
func (o stubOrganism) Species() string               { return o.species }
func (stubOrganism) Line() string                    { return "" }
func (stubOrganism) Stage() pluginapi.LifecycleStage { return "" }
func (stubOrganism) CohortID() (string, bool)        { return "", false }
func (o stubOrganism) HousingID() (string, bool) {
	if o.housingID == nil {
		return "", false
	}
	return *o.housingID, true
}
func (stubOrganism) ProtocolID() (string, bool) { return "", false }
func (stubOrganism) ProjectID() (string, bool)  { return "", false }
func (stubOrganism) Attributes() map[string]any { return nil }

// Contextual lifecycle stage accessors
func (stubOrganism) GetCurrentStage() pluginapi.LifecycleStageRef {
	return pluginapi.NewLifecycleStageContext().Adult()
}
func (stubOrganism) IsActive() bool   { return true }
func (stubOrganism) IsRetired() bool  { return false }
func (stubOrganism) IsDeceased() bool { return false }

type stubHousing struct {
	id          string
	environment string
}

func (h stubHousing) ID() string          { return h.id }
func (stubHousing) CreatedAt() time.Time  { return time.Time{} }
func (stubHousing) UpdatedAt() time.Time  { return time.Time{} }
func (stubHousing) Name() string          { return "" }
func (stubHousing) Facility() string      { return "" }
func (stubHousing) Capacity() int         { return 0 }
func (h stubHousing) Environment() string { return h.environment }

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
	var habitat pluginapi.Rule
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
		organisms: []pluginapi.OrganismView{
			stubOrganism{id: "O1", species: "FrogX", housingID: &housingID},
		},
		housing: []pluginapi.HousingUnitView{
			stubHousing{id: housingID, environment: "dry"},
		},
	}
	res, err := habitat.Evaluate(context.Background(), view, nil)
	if err != nil || len(res.Violations()) != 1 {
		t.Fatalf("expected 1 violation: %+v err=%v", res, err)
	}
}
