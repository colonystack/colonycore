package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"colonycore/pkg/domain"
)

func TestHousingUnit_UpdateAndBranches(t *testing.T) {
	eng := NewDefaultRulesEngine()
	store := NewMemoryStore(eng)
	var facility domain.Facility
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		f, err := tx.CreateFacility(domain.Facility{Name: "Facility-A"})
		if err != nil {
			return err
		}
		facility = f
		return nil
	}); err != nil {
		t.Fatalf("create facility: %v", err)
	}
	// Create valid housing unit
	var created domain.HousingUnit
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "HU-A", FacilityID: facility.ID, Capacity: 2})
		if err != nil {
			return err
		}
		created = h
		return nil
	}); err != nil {
		t.Fatalf("create txn: %v", err)
	}
	// Sleep to ensure updatedAt will differ after second transaction
	time.Sleep(5 * time.Millisecond)
	// Successful update path
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		updated, err := tx.UpdateHousingUnit(created.ID, func(h *domain.HousingUnit) error {
			h.Capacity = 3
			h.Name = "HU-A2"
			return nil
		})
		if err != nil {
			return err
		}
		if updated.Capacity != 3 || updated.Name != "HU-A2" {
			t.Fatalf("unexpected updated %+v", updated)
		}
		if !updated.UpdatedAt.After(created.UpdatedAt) {
			t.Fatalf("expected UpdatedAt to advance")
		}
		return nil
	}); err != nil {
		t.Fatalf("update txn: %v", err)
	}
	// Mutator error branch
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, uerr := tx.UpdateHousingUnit(created.ID, func(_ *domain.HousingUnit) error { return errors.New("boom") })
		if uerr == nil {
			t.Fatalf("expected mutator error")
		}
		return nil
	}); err != nil {
		t.Fatalf("mutator txn: %v", err)
	}
	// Not found branch
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if _, nfErr := tx.UpdateHousingUnit("missing-id", func(_ *domain.HousingUnit) error { return nil }); nfErr == nil {
			t.Fatalf("expected not found error")
		}
		return nil
	}); err != nil {
		t.Fatalf("not found txn: %v", err)
	}
	// Invalid capacity in update branch
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, badErr := tx.UpdateHousingUnit(created.ID, func(h *domain.HousingUnit) error { h.Capacity = 0; return nil }) // h used inside closure
		if badErr == nil {
			t.Fatalf("expected capacity validation error")
		}
		return nil
	}); err != nil {
		t.Fatalf("invalid capacity txn: %v", err)
	}
	// Invalid capacity on create
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if _, cErr := tx.CreateHousingUnit(domain.HousingUnit{Name: "HU-B", FacilityID: facility.ID, Capacity: 0}); cErr == nil {
			t.Fatalf("expected create capacity error")
		}
		return nil
	}); err != nil {
		t.Fatalf("create invalid txn: %v", err)
	}
	// Delete not found branch
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		if dErr := tx.DeleteHousingUnit("missing-id"); dErr == nil {
			t.Fatalf("expected delete not found error")
		}
		return nil
	}); err != nil {
		t.Fatalf("delete not found txn: %v", err)
	}
}
