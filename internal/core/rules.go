package core

import "colonycore/pkg/domain"

// NewRulesEngine constructs an engine instance.
func NewRulesEngine() *domain.RulesEngine {
	return domain.NewRulesEngine()
}

// NewDefaultRulesEngine builds a rules engine with the built-in policy set.
func NewDefaultRulesEngine() *domain.RulesEngine {
	engine := NewRulesEngine()
	engine.Register(NewHousingCapacityRule())
	engine.Register(NewProtocolSubjectCapRule())
	engine.Register(LineageIntegrityRule())
	engine.Register(LifecycleTransitionRule())
	engine.Register(ProtocolCoverageRule())
	return engine
}
