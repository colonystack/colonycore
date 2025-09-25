package core

import "fmt"

// PostgresStore placeholder – embeds a MemoryStore so it satisfies the
// PersistentStore interface while real implementation is pending.
type PostgresStore struct{ *MemoryStore }

// NewPostgresStore returns a placeholder backed by MemoryStore plus a not-implemented error.
func NewPostgresStore(dsn string, engine *RulesEngine) (*PostgresStore, error) {
	ps := &PostgresStore{MemoryStore: NewMemoryStore(engine)}
	return ps, fmt.Errorf("postgres driver not yet implemented")
}
