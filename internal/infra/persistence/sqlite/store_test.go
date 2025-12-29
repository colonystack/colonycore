package sqlite

import (
	"colonycore/internal/entitymodel/sqlbundle"
	"colonycore/pkg/domain"
	entitymodel "colonycore/pkg/domain/entitymodel"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"strings"
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

func TestApplyEntityModelDDLUsesGeneratedSQLiteBundle(t *testing.T) {
	exec := &recordingSQLiteExec{}
	if err := applyEntityModelDDL(exec); err != nil {
		t.Fatalf("applyEntityModelDDL: %v", err)
	}

	expected := sqlbundle.SplitStatements(sqlbundle.SQLite())
	if len(exec.execs) != len(expected) {
		t.Fatalf("expected %d statements, got %d", len(expected), len(exec.execs))
	}
	for i, stmt := range expected {
		if strings.TrimSpace(exec.execs[i]) != strings.TrimSpace(stmt) {
			t.Fatalf("statement %d mismatch:\nwant: %s\ngot:  %s", i, strings.TrimSpace(stmt), strings.TrimSpace(exec.execs[i]))
		}
	}
}

func TestApplyEntityModelDDLError(t *testing.T) {
	exec := &recordingSQLiteExec{fail: true}
	if err := applyEntityModelDDL(exec); err == nil {
		t.Fatalf("expected ddl exec error")
	}
}

type recordingSQLiteExec struct {
	execs []string
	fail  bool
}

func (r *recordingSQLiteExec) Exec(query string, _ ...any) (sql.Result, error) {
	if r.fail {
		return nil, fmt.Errorf("exec fail")
	}
	r.execs = append(r.execs, query)
	return driver.RowsAffected(1), nil
}
