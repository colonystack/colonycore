package core

import (
	"context"
	"testing"
)

func TestOrganism_UpdateDeleteBranches(t *testing.T) {
	eng := NewDefaultRulesEngine()
	store := NewMemoryStore(eng)
	var org Organism
	// Create & update organism
	if _, err := store.RunInTransaction(context.Background(), func(tx Transaction) error {
		created, err := tx.CreateOrganism(Organism{Species: "frog"})
		if err != nil {
			return err
		}
		org = created
		return nil
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := store.RunInTransaction(context.Background(), func(tx Transaction) error {
		updated, err := tx.UpdateOrganism(org.ID, func(o *Organism) error { o.Species = "frog2"; return nil })
		if err != nil {
			return err
		}
		if updated.Species != "frog2" {
			t.Fatalf("update failed")
		}
		return nil
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	// Update missing
	if _, err := store.RunInTransaction(context.Background(), func(tx Transaction) error {
		if _, uerr := tx.UpdateOrganism("missing", func(o *Organism) error { return nil }); uerr == nil {
			t.Fatalf("expected not found")
		}
		return nil
	}); err != nil {
		t.Fatalf("update missing: %v", err)
	}
	// Delete present
	if _, err := store.RunInTransaction(context.Background(), func(tx Transaction) error { return tx.DeleteOrganism(org.ID) }); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Delete missing
	if _, err := store.RunInTransaction(context.Background(), func(tx Transaction) error {
		if derr := tx.DeleteOrganism("missing"); derr == nil {
			t.Fatalf("expected missing delete error")
		}
		return nil
	}); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}
