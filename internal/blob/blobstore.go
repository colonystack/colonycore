package blob

import (
	"context"
	"errors"
	"io"
	"time"
)

// Driver identifies a concrete blob storage backend implementation.
type Driver string

const (
	DriverFilesystem Driver = "fs"     // local filesystem (default, dev)
	DriverS3         Driver = "s3"     // S3 / MinIO compatible
	DriverMemory     Driver = "memory" // in-memory (tests)
)

// PutOptions specifies optional parameters for Put.
type PutOptions struct {
	ContentType string            // MIME type, optional
	Metadata    map[string]string // User metadata (small, flat key-value)
	// TODO: future: server-side encryption, tagging, ACL
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
	URL          string            `json:"url,omitempty"` // optional presigned URL
}

// BlobStore provides a thin S3-like abstraction used by higher layers.
// Semantics intentionally mirror a minimal subset of S3 so that an S3 / MinIO
// adapter can be nearly 1:1 while a filesystem adapter can emulate them.
type BlobStore interface {
	// Put stores a new blob at key. MUST fail if the key already exists.
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (Info, error)
	// Get retrieves the blob contents and metadata. Returns os.ErrNotExist style error if missing.
	Get(ctx context.Context, key string) (Info, io.ReadCloser, error)
	// Head returns metadata only.
	Head(ctx context.Context, key string) (Info, error)
	// Delete removes a blob. Returns (false, nil) if not found.
	Delete(ctx context.Context, key string) (bool, error)
	// List returns blobs whose key has the provided prefix. Stable ordering by key ascending.
	List(ctx context.Context, prefix string) ([]Info, error)
	// PresignURL returns a time-limited URL for the given key (GET). Implementations may
	// return ErrUnsupported if not available.
	PresignURL(ctx context.Context, key string, opts SignedURLOptions) (string, error)
	// Driver returns the configured backend driver string.
	Driver() Driver
}

// ErrUnsupported is returned when an optional capability is not available.
var ErrUnsupported = errors.New("blobstore: unsupported operation")
