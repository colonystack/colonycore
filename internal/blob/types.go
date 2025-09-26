package blob

import (
	"colonycore/internal/blob/core"
)

type (
	Driver           = core.Driver
	PutOptions       = core.PutOptions
	SignedURLOptions = core.SignedURLOptions
	Info             = core.Info
	Store            = core.Store
)

const (
	DriverFilesystem = core.DriverFilesystem
	DriverS3         = core.DriverS3
	DriverMemory     = core.DriverMemory
)

var ErrUnsupported = core.ErrUnsupported
