package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteStorePersistAndReloadReduced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.db")
	store, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	if _, err := store.RunInTransaction(context.Background(), func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Organism: entitymodel.Organism{Name: "Persist"}})
		return e
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	reloaded, err := NewStore(path, domain.NewRulesEngine())
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := len(reloaded.ListOrganisms()); got != 1 {
		t.Fatalf("expected 1 organism, got %d", got)
	}
}
