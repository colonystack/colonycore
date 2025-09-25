package domain

import "context"

// RuleView provides read-only access to domain entities for rule evaluation.
type RuleView interface {
	ListOrganisms() []Organism
	ListHousingUnits() []HousingUnit
	ListProtocols() []Protocol
	FindOrganism(id string) (Organism, bool)
	FindHousingUnit(id string) (HousingUnit, bool)
}

// Rule defines an evaluation executed within a transaction boundary.
type Rule interface {
	Name() string
	Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error)
}

// RulesEngine orchestrates rule evaluation.
type RulesEngine struct {
	rules []Rule
}

// NewRulesEngine constructs an engine instance.
func NewRulesEngine() *RulesEngine {
	return &RulesEngine{}
}

// Register appends a rule to the engine.
func (e *RulesEngine) Register(rule Rule) {
	e.rules = append(e.rules, rule)
}

// Evaluate executes all registered rules and aggregates their results.
func (e *RulesEngine) Evaluate(ctx context.Context, view RuleView, changes []Change) (Result, error) {
	var combined Result
	for _, rule := range e.rules {
		res, err := rule.Evaluate(ctx, view, changes)
		if err != nil {
			return Result{}, err
		}
		combined.Merge(res)
	}
	return combined, nil
}
