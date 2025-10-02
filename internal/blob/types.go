// Package blob re-exports core blob abstractions for stable external imports.
package blob

import (
	"colonycore/internal/blob/core"
)

type (
	// Driver identifies a blob backend driver.
	Driver = core.Driver
	// PutOptions configures a blob write.
	PutOptions = core.PutOptions
	// SignedURLOptions configures URL pre-signing.
	SignedURLOptions = core.SignedURLOptions
	// Info describes stored blob metadata.
	Info = core.Info
	// Store is the interface for blob storage backends.
	Store = core.Store
)

const (
	// DriverFilesystem is the local filesystem driver.
	DriverFilesystem = core.DriverFilesystem
	// DriverS3 is the S3-compatible driver.
	DriverS3 = core.DriverS3
	// DriverMemory is the in-memory test driver.
	DriverMemory = core.DriverMemory
)

// ErrUnsupported indicates an operation isn't supported by a driver.
var ErrUnsupported = core.ErrUnsupported
