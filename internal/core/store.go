package core

import "colonycore/internal/infra/persistence/memory"

type MemoryStore = memory.Store

// NewMemoryStore constructs an in-memory store backed by the provided rules engine.
func NewMemoryStore(engine *RulesEngine) *MemoryStore {
	return memory.NewStore(engine)
}
