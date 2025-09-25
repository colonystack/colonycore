package core

import "colonycore/internal/persistence/sqlite"

type SQLiteStore = sqlite.SQLiteStore

func NewSQLiteStore(path string, engine *RulesEngine) (*SQLiteStore, error) {
	return sqlite.NewSQLiteStore(path, engine)
}
