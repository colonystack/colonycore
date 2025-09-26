package blob

import (
	"context"

	infraS3 "colonycore/internal/infra/blob/s3"
)

// S3Config re-exports the infra S3 configuration type for backwards compatibility within the internal tree.
type S3Config = infraS3.Config

// NewS3 constructs an S3-backed blob.Store from the provided configuration.
func NewS3(ctx context.Context, cfg S3Config) (Store, error) {
	return infraS3.New(ctx, cfg)
}

// OpenFromEnv constructs an S3 store using environment variables.
func OpenFromEnv(ctx context.Context) (Store, error) {
	return infraS3.OpenFromEnv(ctx)
}

// NewMockS3ForTests exposes the lightweight in-memory mock for cross-package tests.
func NewMockS3ForTests() Store { return infraS3.NewMockForTests() }
