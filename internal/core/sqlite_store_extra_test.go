package core

import (
	"context"
	"testing"

	"colonycore/pkg/domain"
)

// TestSQLiteStore_PersistError forces an error in persist by closing the DB before a transaction commit.
func TestSQLiteStore_PersistError(t *testing.T) {
	engine := NewDefaultRulesEngine()
	store, err := NewSQLiteStore("", engine)
	if err != nil {
		// if sqlite unavailable, skip
		t.Skipf("sqlite not available: %v", err)
	}
	// Close underlying DB to induce error on persist
	_ = store.DB().Close()
	_, err = store.RunInTransaction(context.Background(), func(_ domain.Transaction) error { return nil })
	if err == nil {
		// expect error because persist should fail with closed db
		// This covers the branch where persist returns error.
		// Even if future implementation changes, failing silently would hide issues.
		// So enforce an error.
		// NOTE: if implementation changes to reopen DB automatically, adapt test.
		// For now, require error.
		// Use Fatalf to signal mismatch.
		t.Fatalf("expected error from persist with closed db")
	}
}
