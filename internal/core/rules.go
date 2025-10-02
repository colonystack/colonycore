package core

import "colonycore/pkg/domain"

type (
	// Rule is an alias of domain.Rule representing a validation policy executed in transactions.
	Rule = domain.Rule
	// RuleView is an alias of domain.RuleView providing read-only access to state during evaluation.
	RuleView = domain.RuleView
	// RulesEngine is an alias of domain.RulesEngine coordinating rule registration and evaluation.
	RulesEngine = domain.RulesEngine
)

// NewRulesEngine constructs an engine instance.
func NewRulesEngine() *RulesEngine {
	return domain.NewRulesEngine()
}

// NewDefaultRulesEngine builds a rules engine with the built-in policy set.
func NewDefaultRulesEngine() *RulesEngine {
	engine := NewRulesEngine()
	engine.Register(NewHousingCapacityRule())
	engine.Register(NewProtocolSubjectCapRule())
	return engine
}
