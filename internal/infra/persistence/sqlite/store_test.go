package sqlite

import (
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"path/filepath"
	"testing"
)

const organismsTable = "organisms"

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

func TestSQLiteStoreAppliesEntityModelDDL(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.db"), domain.NewRulesEngine())
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	t.Cleanup(func() { _ = store.DB().Close() })
	var tableName string
	if err := store.DB().QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name= ?", organismsTable).Scan(&tableName); err != nil {
		t.Fatalf("lookup organisms table: %v", err)
	}
	if tableName != organismsTable {
		t.Fatalf("expected organisms table, got %s", tableName)
	}
}
