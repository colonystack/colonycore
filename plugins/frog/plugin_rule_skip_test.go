package frog

import (
	"context"
	"testing"

	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
)

// fakeRuleViewNoHousing tests habitat rule skip branches (no housing id, housing lookup miss, non-frog species).
type fakeRuleViewNoHousing struct {
	orgs    []domain.Organism
	housing []domain.HousingUnit
}

func (v fakeRuleViewNoHousing) ListOrganisms() []domain.Organism       { return v.orgs }
func (v fakeRuleViewNoHousing) ListHousingUnits() []domain.HousingUnit { return v.housing }
func (v fakeRuleViewNoHousing) ListProtocols() []domain.Protocol       { return nil }
func (v fakeRuleViewNoHousing) FindOrganism(_ string) (domain.Organism, bool) {
	return domain.Organism{}, false
}
func (v fakeRuleViewNoHousing) FindHousingUnit(_ string) (domain.HousingUnit, bool) {
	return domain.HousingUnit{}, false
}
func strPtr(s string) *string { return &s }

// TestFrogHabitatRuleSkipBranches ensures rule does not emit violations when skip conditions apply.
func TestFrogHabitatRuleSkipBranches(t *testing.T) {
	rule := frogHabitatRule{}
	if rule.Name() != frogHabitatRuleName {
		t.Fatalf("unexpected rule name: %s", rule.Name())
	}
	view := fakeRuleViewNoHousing{orgs: []domain.Organism{
		{Base: domain.Base{ID: "1"}, Species: "Gecko"},                                 // non-frog species
		{Base: domain.Base{ID: "2"}, Species: "FrogX"},                                 // frog with nil housing id
		{Base: domain.Base{ID: "3"}, Species: "FrogY", HousingID: strPtr("H-missing")}, // housing lookup miss
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
