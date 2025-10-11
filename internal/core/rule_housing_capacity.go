package core

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
)

// NewHousingCapacityRule returns the default in-transaction rule enforcing housing capacity constraints.
func NewHousingCapacityRule() domain.Rule {
	return housingCapacityRule{}
}

type housingCapacityRule struct{}

func (housingCapacityRule) Name() string { return "housing_capacity" }

func (housingCapacityRule) Evaluate(_ context.Context, view domain.RuleView, _ []domain.Change) (domain.Result, error) {
	occupancy := make(map[string]int)
	for _, organism := range view.ListOrganisms() {
		if organism.HousingID == nil {
			continue
		}
		occupancy[*organism.HousingID]++
	}

	res := domain.Result{}
	for _, housing := range view.ListHousingUnits() {
		count := occupancy[housing.ID]
		if count > housing.Capacity {
			res.Violations = append(res.Violations, domain.Violation{
				Rule:     "housing_capacity",
				Severity: domain.SeverityBlock,
				Message:  fmt.Sprintf("housing %s (%s) over capacity: %d/%d occupants", housing.Name, housing.ID, count, housing.Capacity),
				Entity:   domain.EntityHousingUnit,
				EntityID: housing.ID,
			})
		}
	}
	return res, nil
}
