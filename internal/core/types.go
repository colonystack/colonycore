package core

import "colonycore/pkg/domain"

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
