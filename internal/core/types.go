package core

import "colonycore/pkg/domain"

type (
	// EntityType aliases domain.EntityType enumerating core entity kinds.
	EntityType = domain.EntityType
	// LifecycleStage aliases domain.LifecycleStage representing organism developmental stages.
	LifecycleStage = domain.LifecycleStage
	// Severity aliases domain.Severity defining rule violation impact levels.
	Severity = domain.Severity
	// Base aliases domain.Base embedding common entity metadata.
	Base = domain.Base
	// Organism aliases domain.Organism describing a tracked individual.
	Organism = domain.Organism
	// Cohort aliases domain.Cohort grouping organisms for joint operations.
	Cohort = domain.Cohort
	// HousingUnit aliases domain.HousingUnit capturing housing metadata and capacity.
	HousingUnit = domain.HousingUnit
	// BreedingUnit aliases domain.BreedingUnit defining reproductive groupings.
	BreedingUnit = domain.BreedingUnit
	// Procedure aliases domain.Procedure recording scheduled or executed actions.
	Procedure = domain.Procedure
	// Protocol aliases domain.Protocol representing an approval or compliance scope.
	Protocol = domain.Protocol
	// Project aliases domain.Project representing an initiative consuming resources.
	Project = domain.Project
	// Change aliases domain.Change describing a state transition considered by rules.
	Change = domain.Change
	// Action aliases domain.Action enumerating CRUD-like semantic operations.
	Action = domain.Action
	// Violation aliases domain.Violation describing a rule evaluation outcome.
	Violation = domain.Violation
	// Result aliases domain.Result aggregating rule evaluation violations.
	Result = domain.Result
	// RuleViolationError aliases domain.RuleViolationError representing a blocking violation error.
	RuleViolationError = domain.RuleViolationError
)

// Canonical entity type identifiers.
const (
	// EntityOrganism identifies an organism entity.
	EntityOrganism = domain.EntityOrganism
	// EntityCohort identifies a cohort entity.
	EntityCohort = domain.EntityCohort
	// EntityHousingUnit identifies a housing unit entity.
	EntityHousingUnit = domain.EntityHousingUnit
	// EntityBreeding identifies a breeding unit entity.
	EntityBreeding = domain.EntityBreeding
	// EntityProcedure identifies a procedure entity.
	EntityProcedure = domain.EntityProcedure
	// EntityProtocol identifies a protocol entity.
	EntityProtocol = domain.EntityProtocol
	// EntityProject identifies a project entity.
	EntityProject = domain.EntityProject
)

// Organism lifecycle stage identifiers.
const (
	// StagePlanned indicates an organism is planned but not yet present.
	StagePlanned = domain.StagePlanned
	// StageLarva indicates larval/embryo developmental phase.
	StageLarva = domain.StageLarva
	// StageJuvenile indicates juvenile phase prior to adulthood.
	StageJuvenile = domain.StageJuvenile
	// StageAdult indicates adult phase.
	StageAdult = domain.StageAdult
	// StageRetired indicates no longer in active experimental/breeding use.
	StageRetired = domain.StageRetired
	// StageDeceased indicates organism is deceased (terminal state).
	StageDeceased = domain.StageDeceased
)

// Rule severity levels.
const (
	// SeverityBlock represents a blocking violation preventing commit.
	SeverityBlock = domain.SeverityBlock
	// SeverityWarn represents a warning violation logged but not blocking.
	SeverityWarn = domain.SeverityWarn
	// SeverityLog represents an informational rule outcome.
	SeverityLog = domain.SeverityLog
)

// Action semantic operation identifiers.
const (
	// ActionCreate indicates an entity creation operation.
	ActionCreate = domain.ActionCreate
	// ActionUpdate indicates an entity update operation.
	ActionUpdate = domain.ActionUpdate
	// ActionDelete indicates an entity deletion operation.
	ActionDelete = domain.ActionDelete
)
