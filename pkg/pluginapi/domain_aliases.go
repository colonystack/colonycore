// Package pluginapi provides a stable surface for plugin authors by re-exporting
// selected domain concepts and rule evaluation primitives.
package pluginapi

import "colonycore/pkg/domain"

// Rule evaluation and result aliases.
type (
	// Rule is an alias of domain.Rule representing a validation hook.
	Rule = domain.Rule
	// RuleView is an alias of domain.RuleView providing a read-only view to rules.
	RuleView = domain.RuleView
	// Change is an alias of domain.Change describing a mutation considered by rules.
	Change = domain.Change
	// Result is an alias of domain.Result aggregating rule violations.
	Result = domain.Result
	// Violation is an alias of domain.Violation detailing a single rule outcome.
	Violation = domain.Violation
)

// Severity level aliases.
const (
	SeverityBlock = domain.SeverityBlock // Block execution
	SeverityWarn  = domain.SeverityWarn  // Warn but continue
	SeverityLog   = domain.SeverityLog   // Log only
)

// Entity type aliases.
const (
	EntityOrganism    = domain.EntityOrganism
	EntityCohort      = domain.EntityCohort
	EntityHousingUnit = domain.EntityHousingUnit
	EntityBreeding    = domain.EntityBreeding
	EntityProcedure   = domain.EntityProcedure
	EntityProtocol    = domain.EntityProtocol
	EntityProject     = domain.EntityProject
)
