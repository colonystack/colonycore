package domain

import "context"

// Transaction exposes the domain operations that a persistence implementation
// must support within an atomic scope.
type Transaction interface {
	Snapshot() TransactionView
	CreateOrganism(Organism) (Organism, error)
	UpdateOrganism(id string, mutator func(*Organism) error) (Organism, error)
	DeleteOrganism(id string) error
	CreateCohort(Cohort) (Cohort, error)
	UpdateCohort(id string, mutator func(*Cohort) error) (Cohort, error)
	DeleteCohort(id string) error
	CreateHousingUnit(HousingUnit) (HousingUnit, error)
	UpdateHousingUnit(id string, mutator func(*HousingUnit) error) (HousingUnit, error)
	DeleteHousingUnit(id string) error
	CreateBreedingUnit(BreedingUnit) (BreedingUnit, error)
	UpdateBreedingUnit(id string, mutator func(*BreedingUnit) error) (BreedingUnit, error)
	DeleteBreedingUnit(id string) error
	CreateProcedure(Procedure) (Procedure, error)
	UpdateProcedure(id string, mutator func(*Procedure) error) (Procedure, error)
	DeleteProcedure(id string) error
	CreateProtocol(Protocol) (Protocol, error)
	UpdateProtocol(id string, mutator func(*Protocol) error) (Protocol, error)
	DeleteProtocol(id string) error
	CreateProject(Project) (Project, error)
	UpdateProject(id string, mutator func(*Project) error) (Project, error)
	DeleteProject(id string) error
	FindHousingUnit(id string) (HousingUnit, bool)
	FindProtocol(id string) (Protocol, bool)
}

// TransactionView provides read-only access to snapshot data for rules.
type TransactionView interface {
	ListOrganisms() []Organism
	ListHousingUnits() []HousingUnit
	FindOrganism(id string) (Organism, bool)
	FindHousingUnit(id string) (HousingUnit, bool)
	ListProtocols() []Protocol
}

// PersistentStore is a minimal abstraction over durable backends. It mirrors
// the subset of store capabilities used directly by higher layers.
type PersistentStore interface {
	RunInTransaction(ctx context.Context, fn func(Transaction) error) (Result, error)
	View(ctx context.Context, fn func(TransactionView) error) error
	GetOrganism(id string) (Organism, bool)
	ListOrganisms() []Organism
	GetHousingUnit(id string) (HousingUnit, bool)
	ListHousingUnits() []HousingUnit
	ListCohorts() []Cohort
	ListProtocols() []Protocol
	ListProjects() []Project
	ListBreedingUnits() []BreedingUnit
	ListProcedures() []Procedure
}
