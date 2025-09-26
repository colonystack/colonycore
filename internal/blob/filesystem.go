package blob

import (
	"colonycore/internal/infra/blob/fs"
)

// NewFilesystem constructs a filesystem-backed blob.Store rooted at the provided path.
// Returns blob.Store to encourage call sites to depend on the interface instead of
// concrete implementations.
func NewFilesystem(root string) (Store, error) {
	return fs.New(root)
}
