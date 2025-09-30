package memory

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
)

func TestStoreRunInTransactionAndSnapshots(t *testing.T) {
	store := NewStore(nil)
	ctx := context.Background()
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, ok := tx.FindHousingUnit("missing"); ok {
			t.Fatalf("expected missing housing lookup")
		}
		created, err := tx.CreateOrganism(domain.Organism{Name: "Test", Species: "Frog"})
		if err != nil {
			return err
		}
		if created.ID == "" {
			t.Fatalf("expected generated ID")
		}
		view := tx.Snapshot()
		if len(view.ListOrganisms()) != 1 {
			t.Fatalf("snapshot mismatch")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("run transaction: %v", err)
	}
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected persisted organism")
	}
	snapshot := store.ExportState()
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snapshot)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored state")
	}
	if store.RulesEngine() == nil {
		t.Fatalf("expected rules engine")
	}
	if store.NowFunc() == nil {
		t.Fatalf("expected now func")
	}
}

func TestStoreRuleViolation(t *testing.T) {
	store := NewStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	_, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Name: "Fail"})
		return e
	})
	if err == nil {
		t.Fatalf("expected rule violation error")
	}
}

type blockingRule struct{}

func (blockingRule) Name() string { return "block" }

func (blockingRule) Evaluate(ctx context.Context, view domain.RuleView, changes []domain.Change) (domain.Result, error) {
	res := domain.Result{}
	res.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	return res, nil
}

func TestUpdateHousingUnitErrors(t *testing.T) {
	store := NewStore(nil)
	ctx := context.Background()
	_, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateHousingUnit("missing", func(*domain.HousingUnit) error { return nil }); err == nil {
			t.Fatalf("expected missing housing error")
		}
		h, err := tx.CreateHousingUnit(domain.HousingUnit{Name: "Unit", Capacity: 2})
		if err != nil {
			return err
		}
		_, err = tx.UpdateHousingUnit(h.ID, func(unit *domain.HousingUnit) error { return fmt.Errorf("boom") })
		if err == nil {
			t.Fatalf("expected mutator error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}
}
