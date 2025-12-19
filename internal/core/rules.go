package core

import "colonycore/pkg/domain"

// NewRulesEngine constructs an engine instance.
func NewRulesEngine() *domain.RulesEngine {
	return domain.NewRulesEngine()
}

func defaultRules() []domain.Rule {
	return []domain.Rule{
		NewHousingCapacityRule(),
		NewProtocolSubjectCapRule(),
		LineageIntegrityRule(),
		LifecycleTransitionRule(),
		ProtocolCoverageRule(),
	}
}

// NewDefaultRulesEngine builds a rules engine with the built-in policy set.
func NewDefaultRulesEngine() *domain.RulesEngine {
	engine := NewRulesEngine()
	for _, rule := range defaultRules() {
		engine.Register(rule)
	}
	return engine
}
