package frog

import (
	"context"
	"testing"

	"colonycore/pkg/pluginapi"
)

// fakeRuleViewNoHousing tests habitat rule skip branches (no housing id, housing lookup miss, non-frog species).
type fakeRuleViewNoHousing struct {
	orgs    []pluginapi.OrganismView
	housing []pluginapi.HousingUnitView
}

func (v fakeRuleViewNoHousing) ListOrganisms() []pluginapi.OrganismView       { return v.orgs }
func (v fakeRuleViewNoHousing) ListHousingUnits() []pluginapi.HousingUnitView { return v.housing }
func (fakeRuleViewNoHousing) ListFacilities() []pluginapi.FacilityView        { return nil }
func (fakeRuleViewNoHousing) ListTreatments() []pluginapi.TreatmentView       { return nil }
func (fakeRuleViewNoHousing) ListObservations() []pluginapi.ObservationView   { return nil }
func (fakeRuleViewNoHousing) ListSamples() []pluginapi.SampleView             { return nil }
func (v fakeRuleViewNoHousing) ListProtocols() []pluginapi.ProtocolView       { return nil }
func (fakeRuleViewNoHousing) ListPermits() []pluginapi.PermitView             { return nil }
func (fakeRuleViewNoHousing) ListProjects() []pluginapi.ProjectView           { return nil }
func (fakeRuleViewNoHousing) ListSupplyItems() []pluginapi.SupplyItemView     { return nil }
func (v fakeRuleViewNoHousing) FindOrganism(_ string) (pluginapi.OrganismView, bool) {
	return nil, false
}
func (v fakeRuleViewNoHousing) FindHousingUnit(_ string) (pluginapi.HousingUnitView, bool) {
	return nil, false
}
func (fakeRuleViewNoHousing) FindFacility(string) (pluginapi.FacilityView, bool) {
	return nil, false
}
func (fakeRuleViewNoHousing) FindTreatment(string) (pluginapi.TreatmentView, bool) {
	return nil, false
}
func (fakeRuleViewNoHousing) FindObservation(string) (pluginapi.ObservationView, bool) {
	return nil, false
}
func (fakeRuleViewNoHousing) FindSample(string) (pluginapi.SampleView, bool) {
	return nil, false
}
func (fakeRuleViewNoHousing) FindPermit(string) (pluginapi.PermitView, bool) {
	return nil, false
}
func (fakeRuleViewNoHousing) FindSupplyItem(string) (pluginapi.SupplyItemView, bool) {
	return nil, false
}
func strPtr(s string) *string { return &s }

// TestFrogHabitatRuleSkipBranches ensures rule does not emit violations when skip conditions apply.
func TestFrogHabitatRuleSkipBranches(t *testing.T) {
	rule := frogHabitatRule{}
	if rule.Name() != frogHabitatRuleName {
		t.Fatalf("unexpected rule name: %s", rule.Name())
	}
	view := fakeRuleViewNoHousing{orgs: []pluginapi.OrganismView{
		stubOrganism{id: "1", species: "Gecko"},                                 // non-frog species
		stubOrganism{id: "2", species: "FrogX"},                                 // frog with nil housing id
		stubOrganism{id: "3", species: "FrogY", housingID: strPtr("H-missing")}, // housing lookup miss
	}}
	res, err := rule.Evaluate(context.Background(), view, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(res.Violations()) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(res.Violations()))
	}
	// ensure contextual interfaces are accessible
	severities := pluginapi.NewSeverityContext()
	_ = severities.Warn()
}
