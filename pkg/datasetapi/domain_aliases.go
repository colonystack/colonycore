// Package datasetapi re-exports selected domain layer types and constants
// to provide a stable API boundary for dataset template plugins and callers.
package datasetapi

import "colonycore/pkg/domain"

// Type aliases exposing core domain entities for dataset APIs.
type (
	// TransactionView is an alias of domain.TransactionView providing a read-only transactional view.
	TransactionView = domain.TransactionView
	// Transaction is an alias of domain.Transaction enabling domain mutations.
	Transaction = domain.Transaction
	// PersistentStore is an alias of domain.PersistentStore used during dataset execution.
	PersistentStore = domain.PersistentStore
	// Base is an alias of domain.Base embedding common entity metadata.
	Base = domain.Base
	// Organism is an alias of domain.Organism.
	Organism = domain.Organism
	// HousingUnit is an alias of domain.HousingUnit.
	HousingUnit = domain.HousingUnit
	// Protocol is an alias of domain.Protocol.
	Protocol = domain.Protocol
	// Project is an alias of domain.Project.
	Project = domain.Project
	// Cohort is an alias of domain.Cohort.
	Cohort = domain.Cohort
	// BreedingUnit is an alias of domain.BreedingUnit.
	BreedingUnit = domain.BreedingUnit
	// Procedure is an alias of domain.Procedure.
	Procedure = domain.Procedure
	// Result is an alias of domain.Result summarizing rule evaluation.
	Result = domain.Result
)

// Lifecycle stage aliases.
const (
	StagePlanned  = domain.StagePlanned  // Planned but not yet started
	StageLarva    = domain.StageLarva    // Early development
	StageJuvenile = domain.StageJuvenile // Intermediate stage
	StageAdult    = domain.StageAdult    // Mature organism
	StageRetired  = domain.StageRetired  // No longer active
	StageDeceased = domain.StageDeceased // Deceased
)
