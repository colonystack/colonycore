package core

import (
	"context"
	"fmt"
	"os"
)

// StorageDriver identifies a concrete persistent storage implementation.
type StorageDriver string

const (
	StorageMemory   StorageDriver = "memory"   // in-memory only (tests / ephemeral)
	StorageSQLite   StorageDriver = "sqlite"   // embedded sqlite file
	StoragePostgres StorageDriver = "postgres" // PostgreSQL server
)

// PersistentStore is a minimal abstraction over durable backends. It mirrors
// the subset of MemoryStore used directly by higher layers.
type PersistentStore interface {
	RunInTransaction(ctx context.Context, fn func(tx *Transaction) error) (Result, error)
	View(ctx context.Context, fn func(TransactionView) error) error
	GetOrganism(id string) (Organism, bool)
	ListOrganisms() []Organism
	GetHousingUnit(id string) (HousingUnit, bool)
	ListHousingUnits() []HousingUnit
	ListCohorts() []Cohort
	ListProtocols() []Protocol
	ListProjects() []Project
	ListBreedingUnits() []BreedingUnit
	ListProcedures() []Procedure
}

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
		return NewMemoryStore(engine), nil
	case StorageSQLite:
		path := os.Getenv("COLONYCORE_SQLITE_PATH")
		return NewSQLiteStore(path, engine)
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
