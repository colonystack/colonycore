package core

import (
	"context"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
)

func newDatasetPersistentStore(store domain.PersistentStore) datasetapi.PersistentStore {
	if store == nil {
		return nil
	}
	return datasetPersistentStoreAdapter{store: store}
}

type datasetPersistentStoreAdapter struct {
	store domain.PersistentStore
}

var _ datasetapi.PersistentStore = datasetPersistentStoreAdapter{}

func (a datasetPersistentStoreAdapter) View(ctx context.Context, fn func(datasetapi.TransactionView) error) error {
	if fn == nil {
		return a.store.View(ctx, func(domain.TransactionView) error { return nil })
	}
	return a.store.View(ctx, func(view domain.TransactionView) error {
		return fn(datasetTransactionViewAdapter{view: view})
	})
}

func (a datasetPersistentStoreAdapter) GetOrganism(id string) (datasetapi.Organism, bool) {
	organism, ok := a.store.GetOrganism(id)
	if !ok {
		return nil, false
	}
	return facadeOrganismFromDomain(organism), true
}

func (a datasetPersistentStoreAdapter) ListOrganisms() []datasetapi.Organism {
	return facadeOrganismsFromDomain(a.store.ListOrganisms())
}

func (a datasetPersistentStoreAdapter) GetHousingUnit(id string) (datasetapi.HousingUnit, bool) {
	unit, ok := a.store.GetHousingUnit(id)
	if !ok {
		return nil, false
	}
	return facadeHousingUnitFromDomain(unit), true
}

func (a datasetPersistentStoreAdapter) ListHousingUnits() []datasetapi.HousingUnit {
	return facadeHousingUnitsFromDomain(a.store.ListHousingUnits())
}

func (a datasetPersistentStoreAdapter) GetFacility(id string) (datasetapi.Facility, bool) {
	facility, ok := a.store.GetFacility(id)
	if !ok {
		return nil, false
	}
	return facadeFacilityFromDomain(facility), true
}

func (a datasetPersistentStoreAdapter) ListFacilities() []datasetapi.Facility {
	return facadeFacilitiesFromDomain(a.store.ListFacilities())
}

func (a datasetPersistentStoreAdapter) ListCohorts() []datasetapi.Cohort {
	return facadeCohortsFromDomain(a.store.ListCohorts())
}

func (a datasetPersistentStoreAdapter) ListTreatments() []datasetapi.Treatment {
	return facadeTreatmentsFromDomain(a.store.ListTreatments())
}

func (a datasetPersistentStoreAdapter) ListObservations() []datasetapi.Observation {
	return facadeObservationsFromDomain(a.store.ListObservations())
}

func (a datasetPersistentStoreAdapter) ListSamples() []datasetapi.Sample {
	return facadeSamplesFromDomain(a.store.ListSamples())
}

func (a datasetPersistentStoreAdapter) ListProtocols() []datasetapi.Protocol {
	return facadeProtocolsFromDomain(a.store.ListProtocols())
}

func (a datasetPersistentStoreAdapter) GetPermit(id string) (datasetapi.Permit, bool) {
	permit, ok := a.store.GetPermit(id)
	if !ok {
		return nil, false
	}
	return facadePermitFromDomain(permit), true
}

func (a datasetPersistentStoreAdapter) ListPermits() []datasetapi.Permit {
	return facadePermitsFromDomain(a.store.ListPermits())
}

func (a datasetPersistentStoreAdapter) ListProjects() []datasetapi.Project {
	return facadeProjectsFromDomain(a.store.ListProjects())
}

func (a datasetPersistentStoreAdapter) ListBreedingUnits() []datasetapi.BreedingUnit {
	return facadeBreedingUnitsFromDomain(a.store.ListBreedingUnits())
}

func (a datasetPersistentStoreAdapter) ListProcedures() []datasetapi.Procedure {
	return facadeProceduresFromDomain(a.store.ListProcedures())
}

func (a datasetPersistentStoreAdapter) ListSupplyItems() []datasetapi.SupplyItem {
	return facadeSupplyItemsFromDomain(a.store.ListSupplyItems())
}

type datasetTransactionViewAdapter struct {
	view domain.TransactionView
}

var _ datasetapi.TransactionView = datasetTransactionViewAdapter{}

func (a datasetTransactionViewAdapter) ListOrganisms() []datasetapi.Organism {
	return facadeOrganismsFromDomain(a.view.ListOrganisms())
}

func (a datasetTransactionViewAdapter) ListHousingUnits() []datasetapi.HousingUnit {
	return facadeHousingUnitsFromDomain(a.view.ListHousingUnits())
}

func (a datasetTransactionViewAdapter) ListFacilities() []datasetapi.Facility {
	return facadeFacilitiesFromDomain(a.view.ListFacilities())
}

func (a datasetTransactionViewAdapter) ListTreatments() []datasetapi.Treatment {
	return facadeTreatmentsFromDomain(a.view.ListTreatments())
}

func (a datasetTransactionViewAdapter) ListObservations() []datasetapi.Observation {
	return facadeObservationsFromDomain(a.view.ListObservations())
}

func (a datasetTransactionViewAdapter) ListSamples() []datasetapi.Sample {
	return facadeSamplesFromDomain(a.view.ListSamples())
}

func (a datasetTransactionViewAdapter) ListProtocols() []datasetapi.Protocol {
	return facadeProtocolsFromDomain(a.view.ListProtocols())
}

func (a datasetTransactionViewAdapter) ListPermits() []datasetapi.Permit {
	return facadePermitsFromDomain(a.view.ListPermits())
}

func (a datasetTransactionViewAdapter) ListProjects() []datasetapi.Project {
	return facadeProjectsFromDomain(a.view.ListProjects())
}

func (a datasetTransactionViewAdapter) ListSupplyItems() []datasetapi.SupplyItem {
	return facadeSupplyItemsFromDomain(a.view.ListSupplyItems())
}

func (a datasetTransactionViewAdapter) FindOrganism(id string) (datasetapi.Organism, bool) {
	organism, ok := a.view.FindOrganism(id)
	if !ok {
		return nil, false
	}
	return facadeOrganismFromDomain(organism), true
}

func (a datasetTransactionViewAdapter) FindHousingUnit(id string) (datasetapi.HousingUnit, bool) {
	unit, ok := a.view.FindHousingUnit(id)
	if !ok {
		return nil, false
	}
	return facadeHousingUnitFromDomain(unit), true
}

func (a datasetTransactionViewAdapter) FindFacility(id string) (datasetapi.Facility, bool) {
	facility, ok := a.view.FindFacility(id)
	if !ok {
		return nil, false
	}
	return facadeFacilityFromDomain(facility), true
}

func (a datasetTransactionViewAdapter) FindTreatment(id string) (datasetapi.Treatment, bool) {
	treatment, ok := a.view.FindTreatment(id)
	if !ok {
		return nil, false
	}
	return facadeTreatmentFromDomain(treatment), true
}

func (a datasetTransactionViewAdapter) FindObservation(id string) (datasetapi.Observation, bool) {
	observation, ok := a.view.FindObservation(id)
	if !ok {
		return nil, false
	}
	return facadeObservationFromDomain(observation), true
}

func (a datasetTransactionViewAdapter) FindSample(id string) (datasetapi.Sample, bool) {
	sample, ok := a.view.FindSample(id)
	if !ok {
		return nil, false
	}
	return facadeSampleFromDomain(sample), true
}

func (a datasetTransactionViewAdapter) FindPermit(id string) (datasetapi.Permit, bool) {
	permit, ok := a.view.FindPermit(id)
	if !ok {
		return nil, false
	}
	return facadePermitFromDomain(permit), true
}

func (a datasetTransactionViewAdapter) FindSupplyItem(id string) (datasetapi.SupplyItem, bool) {
	item, ok := a.view.FindSupplyItem(id)
	if !ok {
		return nil, false
	}
	return facadeSupplyItemFromDomain(item), true
}
