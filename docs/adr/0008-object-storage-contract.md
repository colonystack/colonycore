# ADR-0008: Object Storage Contract

## Status
Accepted (proposed in v0.1 timeframe to unblock dataset export lifecycle and future plugin artifacts)

## Context
Dataset exports (see `internal/adapters/datasets/exporter.go`) currently define an `ObjectStore` interface with only a `Put` method. This limits testability, observability, and future integration with real object storage backends (e.g., S3, GCS, MinIO) or on-prem content-addressable stores. Additionally, downstream needs have emerged:

- Retrieve artifact payloads or signed URLs after creation (UI & API consumption)
- Enumerate artifacts for a given export, template, scope, or retention policy sweep
- Delete artifacts explicitly for governance / retention / user-driven cleanup
- Support future streaming or range reads without changing high-level service layers

The absence of a documented contract prevents consistent implementation across potential backends and complicates acceptance criteria for adding a production-grade store.

## Decision
Formalize an expanded `ObjectStore` interface with explicit CRUD+List semantics while preserving backward compatibility for in-memory tests. The interface emphasizes metadata-first returns (size, content type, created time, optional URL) and defers streaming optimizations until a later ADR.

Proposed interface (Go):

```go
// ObjectStore persists immutable artifact objects addressed by key.
type ObjectStore interface {
    // Put stores a new object. Keys are caller-provided and MUST be collision-free; 
    // implementations SHOULD fail if key already exists unless overwrite is explicitly allowed later.
    Put(ctx context.Context, key string, payload []byte, contentType string, metadata map[string]any) (ExportArtifact, error)

    // Get retrieves the raw bytes and associated artifact metadata. Implementations MAY
    // return a signed URL and nil bytes if using deferred download (future optimization),
    // but for now both metadata and bytes are returned for simplicity.
    Get(ctx context.Context, key string) (ExportArtifact, []byte, error)

    // Delete removes an object. MUST be idempotent: deleting a missing key returns (false, nil).
    Delete(ctx context.Context, key string) (bool, error)

    // List returns keys (and lightweight metadata) under a logical prefix or exact key if provided.
    // Pagination is deliberately omitted in v1; callers must tolerate full scans for test/dev scale.
    List(ctx context.Context, prefix string) ([]ExportArtifact, error)
}
```

Key properties:
- Objects are immutable once stored (no update in place). Mutability would complicate audit trails and caching.
- `ExportArtifact` continues to be the canonical metadata shape.
- `URL` in `ExportArtifact` is optional; for in-memory store we supply a synthetic URL for parity.
- Metadata maps are defensively copied by implementations.

## Rationale
- Completeness: Enables lifecycle management (creation, retrieval, enumeration, deletion).
- Testability: Enables deterministic assertions in unit tests for dataset export flows and retention logic.
- Extensibility: Leaves room for future streaming (`GetReader`), multipart uploads, server-side encryption options without breaking the base interface.
- Simplicity: Single interface remains small; no pagination until real scale demands it (avoid premature abstraction).

## Alternatives Considered
1. Keep single `Put` method and push retrieval to another service layer. Rejected: artificial coupling and more mocks.
2. Introduce a streaming-first interface now. Rejected: YAGNI; complicates in-memory store.
3. Add a generic `Do(operation string, args ...any)` extension point. Rejected: opaque, weakly typed.

## Implications
- `internal/adapters/datasets/exporter.go` will be updated to consume only `Put` for now, but downstream services can start using `Get`/`List` for additional API endpoints (future story: expose artifact download endpoint).
- `MemoryObjectStore` must store payload bytes internally to support `Get`.
- A follow-up ADR will be needed for retention & lifecycle policies (time-to-live, size quotas, encryption, region replication).

## Security & Governance
- Deletion semantics must ensure no partial metadata remnants.
- Future backends must ensure keys are normalized to prevent path traversal (`..`, absolute paths) if mapped to filesystem or bucket object keys.
- Signed URL generation (where applicable) should enforce short-lived expirations; out of scope here.

## Open Questions / Future Work
- Pagination and filtering (content type, created before/after)
- Content hashing (ETag) for integrity and deduplication
- Streaming uploads/downloads
- Server-side encryption configuration
- Lifecycle/retention policies and audit events for deletions
- Multi-tenant namespace isolation

## Acceptance Criteria
- Interface defined as above.
- In-memory implementation supports all methods.
- Unit tests cover: Put/Get roundtrip, List prefix filtering, Delete idempotency, immutability of returned metadata, and that modifying external metadata maps does not mutate stored state.

## References
- Existing snapshot persistence ADR: `0007-storage-baseline.md`
- Dataset export worker: `internal/adapters/datasets/exporter.go`
- Inspired by simplified subsets of AWS S3, GCS, MinIO client patterns.

## Driver Matrix & Configuration
Object storage drivers are runtime-selectable via environment variables; unset values favor a local filesystem implementation for developer ergonomics.

### Drivers
- `fs` (default): Stores blobs on the local filesystem at `COLONYCORE_BLOB_FS_ROOT` (`./blobdata` if unset). Metadata sidecars (`.meta` files) preserve headers without introducing a separate database.
- `s3`: Targets AWS S3 or any S3-compatible endpoint (MinIO, Ceph RGW). Requires an explicit bucket and optional endpoint overrides for on-prem deployments.
- `memory`: Ephemeral in-memory driver used exclusively for tests.

### Environment Variables
- `COLONYCORE_BLOB_DRIVER`: `fs` | `s3` | `memory`; defaults to `fs`.
- `COLONYCORE_BLOB_FS_ROOT`: Directory root for the filesystem driver (`./blobdata` by default). Namespace segregation is handled via caller-provided keys.
- `COLONYCORE_BLOB_S3_BUCKET`: Required bucket name for the S3 driver.
- `COLONYCORE_BLOB_S3_REGION`: Optional AWS region (defaults to `us-east-1`).
- `COLONYCORE_BLOB_S3_ENDPOINT`: Optional custom endpoint URL for S3-compatible services.
- `COLONYCORE_BLOB_S3_PATH_STYLE`: Set to `true` to force path-style access (commonly necessary for MinIO).
- AWS credentials follow the upstream SDK resolution chain (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, shared config files, IAM roles, etc.).

### Semantics & Operator Notes
- `Put` fails if a key already exists; callers delete first to overwrite.
- Keys are opaque strings; `List` applies conventional prefix matching without pagination.
- `PresignURL` returns synthetic local URLs for the filesystem driver and genuine signed GET URLs for S3 backends.
- Metadata (`map[string]string`) is stored as a flat map; large structured metadata belongs in persistent domain stores with blob keys as references.
- The memory driver omits presigning support (`ErrUnsupported`).

### Example Configuration (Postgres + MinIO)
External deployments often pair the filesystem replacement with Postgres for persistent state:

```bash
export COLONYCORE_STORAGE_DRIVER=postgres
export COLONYCORE_POSTGRES_DSN='postgres://colonycore:colonycore@localhost:5432/colonycore?sslmode=disable'

export COLONYCORE_BLOB_DRIVER=s3
export COLONYCORE_BLOB_S3_BUCKET=colonycore-dev
export COLONYCORE_BLOB_S3_REGION=us-east-1
export COLONYCORE_BLOB_S3_ENDPOINT=http://localhost:9000
export COLONYCORE_BLOB_S3_PATH_STYLE=true
```

Local development can omit all variables and rely on SQLite plus the filesystem blob store (`./blobdata`), ensuring zero-config durability for iterative work.
