package blob

import (
	memorystore "colonycore/internal/infra/blob/memory"
)

// NewMemory returns an in-memory blob.Store suitable for tests.
func NewMemory() Store { return memorystore.New() }
