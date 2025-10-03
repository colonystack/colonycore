package frog

import (
	"context"
	"testing"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
)

// fakeRuleViewNoHousing tests habitat rule skip branches (no housing id, housing lookup miss, non-frog species).
type fakeRuleViewNoHousing struct {
	orgs    []datasetapi.Organism
	housing []datasetapi.HousingUnit
}

func (v fakeRuleViewNoHousing) ListOrganisms() []datasetapi.Organism       { return v.orgs }
func (v fakeRuleViewNoHousing) ListHousingUnits() []datasetapi.HousingUnit { return v.housing }
func (v fakeRuleViewNoHousing) ListProtocols() []datasetapi.Protocol       { return nil }
func (v fakeRuleViewNoHousing) FindOrganism(_ string) (datasetapi.Organism, bool) {
	return datasetapi.Organism{}, false
}
func (v fakeRuleViewNoHousing) FindHousingUnit(_ string) (datasetapi.HousingUnit, bool) {
	return datasetapi.HousingUnit{}, false
}
func strPtr(s string) *string { return &s }

// TestFrogHabitatRuleSkipBranches ensures rule does not emit violations when skip conditions apply.
func TestFrogHabitatRuleSkipBranches(t *testing.T) {
	rule := frogHabitatRule{}
	if rule.Name() != frogHabitatRuleName {
		t.Fatalf("unexpected rule name: %s", rule.Name())
	}
	view := fakeRuleViewNoHousing{orgs: []datasetapi.Organism{
		{Base: datasetapi.Base{ID: "1"}, Species: "Gecko"},                                 // non-frog species
		{Base: datasetapi.Base{ID: "2"}, Species: "FrogX"},                                 // frog with nil housing id
		{Base: datasetapi.Base{ID: "3"}, Species: "FrogY", HousingID: strPtr("H-missing")}, // housing lookup miss
	}}
	res, err := rule.Evaluate(context.Background(), view, nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(res.Violations))
	}
	// ensure alias types still accessible
	_ = pluginapi.SeverityWarn
}
