package core

import "colonycore/pkg/domain"

type (
	Rule        = domain.Rule
	RuleView    = domain.RuleView
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
