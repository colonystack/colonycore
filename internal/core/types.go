package core

import "colonycore/pkg/domain"

type (
	EntityType         = domain.EntityType
	LifecycleStage     = domain.LifecycleStage
	Severity           = domain.Severity
	Base               = domain.Base
	Organism           = domain.Organism
	Cohort             = domain.Cohort
	HousingUnit        = domain.HousingUnit
	BreedingUnit       = domain.BreedingUnit
	Procedure          = domain.Procedure
	Protocol           = domain.Protocol
	Project            = domain.Project
	Change             = domain.Change
	Action             = domain.Action
	Violation          = domain.Violation
	Result             = domain.Result
	RuleViolationError = domain.RuleViolationError
)

const (
	EntityOrganism    = domain.EntityOrganism
	EntityCohort      = domain.EntityCohort
	EntityHousingUnit = domain.EntityHousingUnit
	EntityBreeding    = domain.EntityBreeding
	EntityProcedure   = domain.EntityProcedure
	EntityProtocol    = domain.EntityProtocol
	EntityProject     = domain.EntityProject
)

const (
	StagePlanned  = domain.StagePlanned
	StageLarva    = domain.StageLarva
	StageJuvenile = domain.StageJuvenile
	StageAdult    = domain.StageAdult
	StageRetired  = domain.StageRetired
	StageDeceased = domain.StageDeceased
)

const (
	SeverityBlock = domain.SeverityBlock
	SeverityWarn  = domain.SeverityWarn
	SeverityLog   = domain.SeverityLog
)

const (
	ActionCreate = domain.ActionCreate
	ActionUpdate = domain.ActionUpdate
	ActionDelete = domain.ActionDelete
)
