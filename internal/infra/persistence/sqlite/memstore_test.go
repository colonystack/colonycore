package sqlite

import (
	"colonycore/pkg/domain"
	"context"
	"fmt"
	"testing"
	"time"
)

// Migrated minimal representative tests; original exhaustive tests remain at old path until cleanup.

func TestMemStoreBasicLifecycle(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if store.NowFunc() == nil {
		t.Fatalf("expected NowFunc to be initialized")
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, err := tx.CreateOrganism(domain.Organism{Name: "Specimen", Species: "Test"})
		return err
	}); err != nil {
		t.Fatalf("create organism: %v", err)
	}
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected 1 organism")
	}
	snapshot := store.ExportState()
	store.ImportState(Snapshot{})
	if len(store.ListOrganisms()) != 0 {
		t.Fatalf("expected cleared state")
	}
	store.ImportState(snapshot)
	if len(store.ListOrganisms()) != 1 {
		t.Fatalf("expected restored organism")
	}
}

func TestMemStoreRuleViolation(t *testing.T) {
	store := newMemStore(domain.NewRulesEngine())
	store.RulesEngine().Register(blockingRule{})
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Name: "Fail"})
		return e
	}); err == nil {
		t.Fatalf("expected violation error")
	}
}

type blockingRule struct{}

func (blockingRule) Name() string { return "block" }
func (blockingRule) Evaluate(_ context.Context, _ domain.RuleView, _ []domain.Change) (domain.Result, error) {
	r := domain.Result{}
	r.Merge(domain.Result{Violations: []domain.Violation{{Rule: "block", Severity: domain.SeverityBlock}}})
	return r, nil
}

func TestMemStoreCRUDReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	var projectID string
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		proj, err := tx.CreateProject(domain.Project{Code: "PRJ", Title: "Project"})
		if err != nil {
			return err
		}
		projectID = proj.ID
		if _, err := tx.CreateOrganism(domain.Organism{Name: "Alpha", Species: "Frog"}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if got := len(store.ListProjects()); got != 1 {
		t.Fatalf("expected 1 project, got %d", got)
	}
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		if _, err := tx.UpdateProject(projectID, func(p *domain.Project) error { p.Description = "updated"; return nil }); err != nil {
			return err
		}
		return tx.DeleteProject(projectID)
	}); err != nil {
		t.Fatalf("mutate: %v", err)
	}
}

func TestMemStoreProcedureLifecycleReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		prot, err := tx.CreateProtocol(domain.Protocol{Code: "P", Title: "Proto", MaxSubjects: 5})
		if err != nil {
			return err
		}
		_, err = tx.CreateProcedure(domain.Procedure{Name: "Check", Status: "scheduled", ScheduledAt: now, ProtocolID: prot.ID})
		return err
	}); err != nil {
		t.Fatalf("create procedure: %v", err)
	}
	if got := len(store.ListProcedures()); got != 1 {
		t.Fatalf("expected one procedure, got %d", got)
	}
}

func TestMemStoreViewSnapshotReduced(t *testing.T) {
	store := newMemStore(nil)
	ctx := context.Background()
	if err := store.View(ctx, func(v domain.TransactionView) error {
		if len(v.ListOrganisms()) != 0 {
			return fmt.Errorf("expected empty")
		}
		return nil
	}); err != nil {
		t.Fatalf("view: %v", err)
	}
}
