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

func (a datasetPersistentStoreAdapter) ListCohorts() []datasetapi.Cohort {
	return facadeCohortsFromDomain(a.store.ListCohorts())
}

func (a datasetPersistentStoreAdapter) ListProtocols() []datasetapi.Protocol {
	return facadeProtocolsFromDomain(a.store.ListProtocols())
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

func (a datasetTransactionViewAdapter) ListProtocols() []datasetapi.Protocol {
	return facadeProtocolsFromDomain(a.view.ListProtocols())
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
