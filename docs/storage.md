# Storage

ColonyCore now supports a minimal durable storage baseline.

## At a Glance

All storage drivers (persistent state and blob/object) are selected exclusively via environment variables—no code changes or recompilation required. Unset variables fall back to safe local defaults (SQLite file + filesystem blob store). This enables frictionless local development while allowing production deployments to promote to Postgres and S3/MinIO by configuration only.

Quick reference:

Persistent store drivers:
* (default) unset -> sqlite (`./colonycore.db`, override with `COLONYCORE_SQLITE_PATH`)
* `COLONYCORE_STORAGE_DRIVER=memory` -> in-memory (ephemeral)
* `COLONYCORE_STORAGE_DRIVER=sqlite` -> explicit sqlite
* `COLONYCORE_STORAGE_DRIVER=postgres` -> requires `COLONYCORE_POSTGRES_DSN`

Blob store drivers:
* (default) unset or `fs` -> local filesystem (`./blobdata`, override with `COLONYCORE_BLOB_FS_ROOT`)
* `s3` -> AWS S3 or S3-compatible (MinIO, Ceph RGW, etc.)
	- Required: `COLONYCORE_BLOB_S3_BUCKET`
	- Optional: `COLONYCORE_BLOB_S3_REGION`, `COLONYCORE_BLOB_S3_ENDPOINT` (for MinIO), `COLONYCORE_BLOB_S3_PATH_STYLE=true` (often needed for MinIO)
* `memory` -> in-memory (ephemeral, tests)

Example (Postgres + MinIO):

```bash
export COLONYCORE_STORAGE_DRIVER=postgres
export COLONYCORE_POSTGRES_DSN='postgres://colonycore:colonycore@localhost:5432/colonycore?sslmode=disable'

export COLONYCORE_BLOB_DRIVER=s3
export COLONYCORE_BLOB_S3_BUCKET=colonycore-dev
export COLONYCORE_BLOB_S3_REGION=us-east-1
export COLONYCORE_BLOB_S3_ENDPOINT=http://localhost:9000
export COLONYCORE_BLOB_S3_PATH_STYLE=true
```

## Drivers (See ADR-0007 for rationale)

* memory (default in tests): purely in-memory, non-durable.
* sqlite (default for local development): embedded file `colonycore.db` in the current working directory.
* postgres (experimental placeholder): planned for higher concurrency requirements.

## Configuration

Environment variables:

* `COLONYCORE_STORAGE_DRIVER` – one of `memory`, `sqlite`, `postgres`. Defaults to `sqlite` when unset (except test helpers still use memory directly).
* `COLONYCORE_SQLITE_PATH` – path to the sqlite DB file (default: `./colonycore.db`).
* `COLONYCORE_POSTGRES_DSN` – connection string when using postgres.

## SQLite Implementation Notes

The initial implementation snapshots the entire in-memory state after each successful transaction into a single table (`state`) storing JSON blobs per entity bucket. This keeps the code surface small while providing durability. It is not yet optimized for large datasets or high write throughput.

Concurrency: The snapshot approach serializes writes (same as the in-memory store) but allows multiple readers. For higher concurrency (parallel writers, row-level locking) use the future Postgres driver once implemented.

Future enhancements can incrementally introduce:

1. Normalized tables & indexes.
2. Incremental (per-entity) persistence instead of whole snapshot writes.
3. Background checkpointing and write batching.
4. Migration framework & versioned schema.

## Postgres Roadmap

Planned work includes a normalized schema, row-level locking and transactional consistency shared directly with the rules engine. For now the driver is a placeholder so configuration and documentation are aligned early.

## Blob Store

ColonyCore provides a thin, S3-compatible blob abstraction used for exporting dataset artifacts or other binary payloads. It is intentionally minimal (Put, Get, Head, Delete, List, Presign) to keep adapters simple.

### Drivers

* `fs` (default): local filesystem rooted at `COLONYCORE_BLOB_FS_ROOT` (default `./blobdata`). Suitable for development and single-node deployments.
* `s3`: AWS S3 or any S3-compatible system (MinIO, Ceph RGW, etc.).
* `memory`: non-durable, in-memory (tests only).

### Environment Configuration

General:

* `COLONYCORE_BLOB_DRIVER` – `fs` | `s3` | `memory` (default `fs`).

Filesystem driver specifics:

* `COLONYCORE_BLOB_FS_ROOT` – directory root for object data (default `./blobdata`). Each blob becomes a regular file; metadata is stored in a sibling JSON sidecar with `.meta` extension.

S3 driver specifics:

* `COLONYCORE_BLOB_S3_BUCKET` – (required) target bucket name.
* `COLONYCORE_BLOB_S3_REGION` – AWS region (default `us-east-1`).
* `COLONYCORE_BLOB_S3_ENDPOINT` – optional custom endpoint URL (e.g. `http://localhost:9000` for MinIO).
* `COLONYCORE_BLOB_S3_PATH_STYLE` – `true` to force path-style addressing (commonly needed for MinIO); default `false`.
* AWS credentials resolved via the standard SDK chain (env vars: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, shared config/credentials files, IAM roles, etc.). If a custom endpoint is used (MinIO), set explicit credentials via env vars or the endpoint's access policies.

### Semantics & Notes

* `Put` fails if the key already exists (no overwrite). Implement overwrite explicitly by deleting first if desired.
* The Go API exposes all drivers through the `blob.Store` interface, keeping higher layers decoupled from concrete backends (e.g. `blob.NewFilesystem`, `blob.NewMockS3ForTests`).
* Keys are treated as opaque strings with conventional prefix semantics for `List(prefix)`.
* Filesystem adapter returns opaque `http://local.blob/<key>` style URLs for local development; these are not protected and are only hints (no server is started by the library).
* `PresignURL` is currently implemented for `fs` (dummy local URL) and real presigned GET for `s3`.
* Metadata is a flat `map[string]string`. Large or structured metadata should be stored in domain storage and referenced by key.
* The `memory` driver does not support presigning (returns `ErrUnsupported`).

### Future Enhancements

Potential future extensions (deferred until necessary):

1. Server-side encryption flags (SSE-S3 / SSE-KMS).
2. Multi-part uploads (exposed only if needed for very large artifacts).
3. Tagging, object lifecycle / retention policies.
4. Streaming / ranged GET support (pass-through from S3).

### Minimal Usage Example (S3)

```bash
export COLONYCORE_BLOB_DRIVER=s3
export COLONYCORE_BLOB_S3_BUCKET=my-colonycore
export COLONYCORE_BLOB_S3_REGION=us-east-1
# For MinIO/local:
# export COLONYCORE_BLOB_S3_ENDPOINT=http://localhost:9000
# export COLONYCORE_BLOB_S3_PATH_STYLE=true
```

Then in Go code:

```go
store, err := blob.Open(context.Background())
if err != nil { panic(err) }
_, _ = store.Put(ctx, "exports/report.json", bytes.NewReader(data), blob.PutOptions{ContentType: "application/json"})
```
