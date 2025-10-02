// Package core defines core abstractions for blob storage backends
// used internally by higher-level services.
package core

import (
	"context"
	"errors"
	"io"
	"time"
)

// Driver identifies a concrete blob storage backend implementation.
type Driver string

const (
	// DriverFilesystem represents the local filesystem implementation.
	DriverFilesystem Driver = "fs" // local filesystem (default, dev)
	// DriverS3 represents an S3 / MinIO compatible implementation.
	DriverS3 Driver = "s3" // S3 / MinIO compatible
	// DriverMemory represents an in-memory implementation typically used in tests.
	DriverMemory Driver = "memory" // in-memory (tests)
)

// PutOptions specifies optional parameters for Put.
type PutOptions struct {
	ContentType string            // MIME type, optional
	Metadata    map[string]string // User metadata (small, flat key-value)
}

// SignedURLOptions holds options for generating a pre-signed URL.
type SignedURLOptions struct {
	Method  string        // GET|PUT (currently only GET used internally)
	Expiry  time.Duration // default 15m
	Headers map[string]string
}

// Info describes a stored blob.
type Info struct {
	Key          string            `json:"key"`
	Size         int64             `json:"size_bytes"`
	ContentType  string            `json:"content_type,omitempty"`
	ETag         string            `json:"etag,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	LastModified time.Time         `json:"last_modified"`
	URL          string            `json:"url,omitempty"`
}

// Store provides a thin S3-like abstraction used by higher layers.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (Info, error)
	Get(ctx context.Context, key string) (Info, io.ReadCloser, error)
	Head(ctx context.Context, key string) (Info, error)
	Delete(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]Info, error)
	PresignURL(ctx context.Context, key string, opts SignedURLOptions) (string, error)
	Driver() Driver
}

// ErrUnsupported is returned when an optional capability is not available.
var ErrUnsupported = errors.New("blobstore: unsupported operation")
