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
	CreateFacility(Facility) (Facility, error)
	UpdateFacility(id string, mutator func(*Facility) error) (Facility, error)
	DeleteFacility(id string) error
	CreateBreedingUnit(BreedingUnit) (BreedingUnit, error)
	UpdateBreedingUnit(id string, mutator func(*BreedingUnit) error) (BreedingUnit, error)
	DeleteBreedingUnit(id string) error
	CreateLine(Line) (Line, error)
	UpdateLine(id string, mutator func(*Line) error) (Line, error)
	DeleteLine(id string) error
	CreateStrain(Strain) (Strain, error)
	UpdateStrain(id string, mutator func(*Strain) error) (Strain, error)
	DeleteStrain(id string) error
	CreateGenotypeMarker(GenotypeMarker) (GenotypeMarker, error)
	UpdateGenotypeMarker(id string, mutator func(*GenotypeMarker) error) (GenotypeMarker, error)
	DeleteGenotypeMarker(id string) error
	CreateProcedure(Procedure) (Procedure, error)
	UpdateProcedure(id string, mutator func(*Procedure) error) (Procedure, error)
	DeleteProcedure(id string) error
	CreateTreatment(Treatment) (Treatment, error)
	UpdateTreatment(id string, mutator func(*Treatment) error) (Treatment, error)
	DeleteTreatment(id string) error
	CreateObservation(Observation) (Observation, error)
	UpdateObservation(id string, mutator func(*Observation) error) (Observation, error)
	DeleteObservation(id string) error
	CreateSample(Sample) (Sample, error)
	UpdateSample(id string, mutator func(*Sample) error) (Sample, error)
	DeleteSample(id string) error
	CreateProtocol(Protocol) (Protocol, error)
	UpdateProtocol(id string, mutator func(*Protocol) error) (Protocol, error)
	DeleteProtocol(id string) error
	CreatePermit(Permit) (Permit, error)
	UpdatePermit(id string, mutator func(*Permit) error) (Permit, error)
	DeletePermit(id string) error
	CreateProject(Project) (Project, error)
	UpdateProject(id string, mutator func(*Project) error) (Project, error)
	DeleteProject(id string) error
	CreateSupplyItem(SupplyItem) (SupplyItem, error)
	UpdateSupplyItem(id string, mutator func(*SupplyItem) error) (SupplyItem, error)
	DeleteSupplyItem(id string) error
	FindHousingUnit(id string) (HousingUnit, bool)
	FindProtocol(id string) (Protocol, bool)
	FindFacility(id string) (Facility, bool)
	FindLine(id string) (Line, bool)
	FindStrain(id string) (Strain, bool)
	FindGenotypeMarker(id string) (GenotypeMarker, bool)
	FindTreatment(id string) (Treatment, bool)
	FindObservation(id string) (Observation, bool)
	FindSample(id string) (Sample, bool)
	FindPermit(id string) (Permit, bool)
	FindSupplyItem(id string) (SupplyItem, bool)
	FindProcedure(id string) (Procedure, bool)
}

// TransactionView provides read-only access to snapshot data for rules.
type TransactionView interface {
	ListOrganisms() []Organism
	ListHousingUnits() []HousingUnit
	ListFacilities() []Facility
	ListLines() []Line
	ListStrains() []Strain
	ListGenotypeMarkers() []GenotypeMarker
	FindOrganism(id string) (Organism, bool)
	FindHousingUnit(id string) (HousingUnit, bool)
	FindFacility(id string) (Facility, bool)
	FindLine(id string) (Line, bool)
	FindStrain(id string) (Strain, bool)
	FindGenotypeMarker(id string) (GenotypeMarker, bool)
	ListTreatments() []Treatment
	ListObservations() []Observation
	ListSamples() []Sample
	ListProtocols() []Protocol
	ListPermits() []Permit
	ListProjects() []Project
	ListSupplyItems() []SupplyItem
	FindTreatment(id string) (Treatment, bool)
	FindObservation(id string) (Observation, bool)
	FindSample(id string) (Sample, bool)
	FindPermit(id string) (Permit, bool)
	FindSupplyItem(id string) (SupplyItem, bool)
	FindProcedure(id string) (Procedure, bool)
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
	GetFacility(id string) (Facility, bool)
	ListFacilities() []Facility
	GetLine(id string) (Line, bool)
	ListLines() []Line
	GetStrain(id string) (Strain, bool)
	ListStrains() []Strain
	GetGenotypeMarker(id string) (GenotypeMarker, bool)
	ListGenotypeMarkers() []GenotypeMarker
	ListCohorts() []Cohort
	ListTreatments() []Treatment
	ListObservations() []Observation
	ListSamples() []Sample
	ListProtocols() []Protocol
	GetPermit(id string) (Permit, bool)
	ListPermits() []Permit
	ListProjects() []Project
	ListBreedingUnits() []BreedingUnit
	ListProcedures() []Procedure
	ListSupplyItems() []SupplyItem
}
