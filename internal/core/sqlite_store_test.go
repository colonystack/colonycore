package core_test

import (
	core "colonycore/internal/core"
	"colonycore/pkg/domain"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteStoreSnapshot(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	store, err := core.NewSQLiteStore(dbPath, core.NewRulesEngine())
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	ctx := context.Background()
	if _, err := store.RunInTransaction(ctx, func(tx domain.Transaction) error {
		_, e := tx.CreateOrganism(domain.Organism{Name: "Alpha", Species: "Frog"})
		return e
	}); err != nil {
		t.Fatalf("create organism: %v", err)
	}
	store2, err := core.NewSQLiteStore(dbPath, core.NewRulesEngine())
	if err != nil {
		t.Fatalf("reopen sqlite store: %v", err)
	}
	list := store2.ListOrganisms()
	if len(list) != 1 || list[0].Name != "Alpha" {
		t.Fatalf("expected snapshot reload, got %+v", list)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("db file missing: %v", err)
	}
}
