package core

import (
	"context"
	"testing"
)

// These supplemental tests exercise a few lower-coverage paths to raise overall threshold.
func TestStore_UpdateHousingUnit_InvalidCapacity(t *testing.T) {
	eng := NewDefaultRulesEngine()
	store := NewMemoryStore(eng)
	_, err := store.RunInTransaction(context.Background(), func(tx Transaction) error {
		h, err := tx.CreateHousingUnit(HousingUnit{Capacity: 1, Name: "A"})
		if err != nil {
			return err
		}
		// mutation that makes capacity invalid
		_, uerr := tx.UpdateHousingUnit(h.ID, func(hp *HousingUnit) error { hp.Capacity = 0; return nil })
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
	_, err := store.RunInTransaction(context.Background(), func(tx Transaction) error {
		if _, bErr := tx.UpdateBreedingUnit("missing", func(b *BreedingUnit) error { return nil }); bErr == nil {
			t.Fatalf("expected not found error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("txn err: %v", err)
	}
}
