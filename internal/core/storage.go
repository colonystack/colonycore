package core

import (
	"colonycore/internal/infra/persistence/memory"
	"colonycore/internal/infra/persistence/sqlite"
	"colonycore/pkg/domain"
	"fmt"
	"os"
)

// StorageDriver identifies a concrete persistent storage implementation.
type StorageDriver string

// Supported storage driver identifiers.
const (
	// StorageMemory provides an in-memory ephemeral store (primarily tests).
	StorageMemory StorageDriver = "memory" // in-memory only (tests / ephemeral)
	// StorageSQLite provides an embedded SQLite-backed store.
	StorageSQLite StorageDriver = "sqlite" // embedded sqlite file
	// StoragePostgres provides a PostgreSQL-backed store.
	StoragePostgres StorageDriver = "postgres" // PostgreSQL server
)

type (
	// Transaction aliases domain.Transaction representing a mutable unit of work.
	Transaction = domain.Transaction
	// TransactionView aliases domain.TransactionView exposing read-only state for observers.
	TransactionView = domain.TransactionView
	// PersistentStore aliases domain.PersistentStore abstracting backing storage implementations.
	PersistentStore = domain.PersistentStore
)

// OpenPersistentStore selects a backend using environment variables.
// Defaults to sqlite when unset.
//
//	COLONYCORE_STORAGE_DRIVER: memory|sqlite|postgres (default sqlite)
//	COLONYCORE_SQLITE_PATH: path to sqlite file (default ./colonycore.db)
//	COLONYCORE_POSTGRES_DSN: postgres DSN when driver=postgres
func OpenPersistentStore(engine *RulesEngine) (PersistentStore, error) {
	driver := os.Getenv("COLONYCORE_STORAGE_DRIVER")
	if driver == "" {
		driver = string(StorageSQLite)
	}
	switch StorageDriver(driver) {
	case StorageMemory:
		return memory.NewStore(engine), nil
	case StorageSQLite:
		path := os.Getenv("COLONYCORE_SQLITE_PATH")
		return sqlite.NewStore(path, engine)
	case StoragePostgres:
		dsn := os.Getenv("COLONYCORE_POSTGRES_DSN")
		ps, err := NewPostgresStore(dsn, engine)
		if err != nil {
			return nil, err
		}
		return ps, nil
	default:
		return nil, fmt.Errorf("unknown storage driver %s", driver)
	}
}
