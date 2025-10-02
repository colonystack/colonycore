package core

import (
	"context"
	"fmt"
)

// NewHousingCapacityRule returns the default in-transaction rule enforcing housing capacity constraints.
func NewHousingCapacityRule() Rule {
	return housingCapacityRule{}
}

type housingCapacityRule struct{}

func (housingCapacityRule) Name() string { return "housing_capacity" }

func (housingCapacityRule) Evaluate(_ context.Context, view RuleView, _ []Change) (Result, error) {
	occupancy := make(map[string]int)
	for _, organism := range view.ListOrganisms() {
		if organism.HousingID == nil {
			continue
		}
		occupancy[*organism.HousingID]++
	}

	res := Result{}
	for _, housing := range view.ListHousingUnits() {
		count := occupancy[housing.ID]
		if count > housing.Capacity {
			res.Violations = append(res.Violations, Violation{
				Rule:     "housing_capacity",
				Severity: SeverityBlock,
				Message:  fmt.Sprintf("housing %s (%s) over capacity: %d/%d occupants", housing.Name, housing.ID, count, housing.Capacity),
				Entity:   EntityHousingUnit,
				EntityID: housing.ID,
			})
		}
	}
	return res, nil
}
