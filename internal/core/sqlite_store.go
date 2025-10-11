package core

import (
	"colonycore/internal/infra/persistence/sqlite"
	"colonycore/pkg/domain"
)

// NewSQLiteStore constructs a new SQLite-backed persistent store using the
// provided file path (may be empty for default) and rules engine.
// Retained wrapper name for backward compatibility; underlying type is sqlite.Store.
func NewSQLiteStore(path string, engine *domain.RulesEngine) (*sqlite.Store, error) {
	return sqlite.NewStore(path, engine)
}
