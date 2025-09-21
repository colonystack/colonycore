package core

import "context"

// Rule defines an evaluation executed within a transaction boundary.
type Rule interface {
	Name() string
	Evaluate(ctx context.Context, view TransactionView, changes []Change) (Result, error)
}

// RulesEngine orchestrates rule evaluation.
type RulesEngine struct {
	rules []Rule
}

// NewRulesEngine constructs an engine instance.
func NewRulesEngine() *RulesEngine {
	return &RulesEngine{}
}

// NewDefaultRulesEngine builds a rules engine with the built-in policy set.
func NewDefaultRulesEngine() *RulesEngine {
	engine := NewRulesEngine()
	engine.Register(NewHousingCapacityRule())
	engine.Register(NewProtocolSubjectCapRule())
	return engine
}

// Register appends a rule to the engine.
func (e *RulesEngine) Register(rule Rule) {
	e.rules = append(e.rules, rule)
}

// Evaluate executes all registered rules and aggregates their results.
func (e *RulesEngine) Evaluate(ctx context.Context, view TransactionView, changes []Change) (Result, error) {
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
