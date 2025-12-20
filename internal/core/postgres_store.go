package core

import (
	"colonycore/internal/infra/persistence/postgres"
	"colonycore/pkg/domain"
)

// NewPostgresStore constructs a Postgres-backed store from the provided DSN.
func NewPostgresStore(dsn string, engine *domain.RulesEngine) (*postgres.Store, error) {
	return postgres.NewStore(dsn, engine)
}
