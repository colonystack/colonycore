package core

import "colonycore/internal/persistence/sqlite"

// NewSQLiteStore constructs a new SQLite-backed persistent store using the
// provided file path (may be empty for default) and rules engine.
// Retained wrapper name for backward compatibility; underlying type is sqlite.Store.
func NewSQLiteStore(path string, engine *RulesEngine) (*sqlite.Store, error) {
	return sqlite.NewStore(path, engine)
}
