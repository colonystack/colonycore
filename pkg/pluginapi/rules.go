package pluginapi

import "context"

// Rule defines a plugin-contributed validation routine executed within a transaction.
type Rule interface {
	Name() string
	Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error)
}

// RuleView exposes read-only access to core entities during rule evaluation.
type RuleView interface {
	ListOrganisms() []OrganismView
	ListHousingUnits() []HousingUnitView
	ListProtocols() []ProtocolView
	FindOrganism(id string) (OrganismView, bool)
	FindHousingUnit(id string) (HousingUnitView, bool)
}
