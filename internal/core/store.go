package core

import "colonycore/internal/infra/persistence/memory"

// NewMemoryStore constructs an in-memory store backed by the provided rules engine.
func NewMemoryStore(engine *RulesEngine) *memory.Store {
	return memory.NewStore(engine)
}
