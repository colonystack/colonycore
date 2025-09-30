package core

import "colonycore/internal/persistence/sqlite"

func NewSQLiteStore(path string, engine *RulesEngine) (*sqlite.SQLiteStore, error) {
	return sqlite.NewSQLiteStore(path, engine)
}
