package core

import (
	"colonycore/internal/infra/persistence/memory"
	"colonycore/pkg/domain"
)

// NewMemoryStore constructs an in-memory store backed by the provided rules engine.
func NewMemoryStore(engine *domain.RulesEngine) *memory.Store {
	return memory.NewStore(engine)
}
