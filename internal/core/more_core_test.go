package core

import (
	"context"
	"testing"

	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
)

// These supplemental tests exercise a few lower-coverage paths to raise overall threshold.
func TestStore_UpdateHousingUnit_InvalidCapacity(t *testing.T) {
	eng := NewDefaultRulesEngine()
	store := NewMemoryStore(eng)
	_, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		f, err := tx.CreateFacility(domain.Facility{Facility: entitymodel.Facility{Name: "F"}})
		if err != nil {
			return err
		}
		h, err := tx.CreateHousingUnit(domain.HousingUnit{HousingUnit: entitymodel.HousingUnit{Capacity: 1, Name: "A", FacilityID: f.ID}})
		if err != nil {
			return err
		}
		// mutation that makes capacity invalid
		_, uerr := tx.UpdateHousingUnit(h.ID, func(hp *domain.HousingUnit) error { hp.Capacity = 0; return nil })
		if uerr == nil {
			t.Fatalf("expected capacity error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("txn err: %v", err)
	}
}

func TestStore_UpdateBreedingUnit_NotFound(t *testing.T) {
	eng := NewDefaultRulesEngine()
	store := NewMemoryStore(eng)
	_, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if _, bErr := tx.UpdateBreedingUnit("missing", func(_ *domain.BreedingUnit) error { return nil }); bErr == nil {
			t.Fatalf("expected not found error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("txn err: %v", err)
	}
}
