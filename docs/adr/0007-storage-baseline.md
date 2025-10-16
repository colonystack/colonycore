# ADR-0007: Storage Baseline (SQLite Snapshot + Postgres Placeholder)

- Status: Accepted
- Date: 2025-09-23
- Deciders: Tobias Harnickell
- Linked RFCs: RFC-0001
- Context: Initially ColonyCore only supported an in-memory `MemoryStore`, limiting durability and multi-process usage. A minimal persistent option was required for local development and early adopters without committing prematurely to a full relational schema and migration surface.

## Decision
Adopt an embedded SQLite-backed store (internally `sqlite.Store`, exposed through the convenience constructor `core.NewSQLiteStore`) as the default persistent driver for local development. It reuses existing transactional logic in `MemoryStore` and after each successful transaction snapshots the entire in-memory state into a single SQLite table (`state`) containing JSON blobs per entity bucket. Provide an environment-based factory (`OpenPersistentStore`) selecting among `memory`, `sqlite`, or `postgres` drivers. Introduce a placeholder `PostgresStore` that currently delegates to `MemoryStore` and returns a not-implemented error, reserving the configuration contract for future high-concurrency deployments.

## Rationale
1. **Speed of implementation**: Snapshotting leverages existing concurrency control and rule evaluation paths with minimal code.
2. **Developer ergonomics**: SQLite is file-based, needs no additional services, and yields durable state between runs.
3. **Incremental path**: The abstraction (`PersistentStore`) allows introducing normalized schemas and incremental write strategies later without breaking service code.
4. **Early configuration stability**: Exposing `COLONYCORE_STORAGE_DRIVER` now prevents later breaking changes in deployment manifests.
5. **Operational simplicity**: Whole-state snapshots reduce migration complexity during the pre-schema phase, deferring index and normalization design until real workload data is available.

## Consequences
### Positive
- Immediate durability for local workflows.
- Clear extension point for future stores (e.g., a fully normalized Postgres backend, object storage event sourcing, etc.).
- Easy rollback: delete the SQLite file to reset state; tests remain fast using memory.

### Negative / Trade-offs
- Snapshot approach writes the entire state for each transaction; inefficient for large datasets.
- No fine-grained concurrency improvements yet; write throughput remains serialized.
- Lacks schema-level constraints/enforcement that a normalized relational design would provide.

### Mitigations / Future Work
| Concern | Planned Mitigation |
| ------- | ------------------ |
| Snapshot performance degradation | Move to per-entity delta persistence + WAL tuning. |
| Concurrency scaling | Implement real Postgres driver with row-level locking and indexes. |
| Schema evolution & migrations | Introduce version table + migration framework once normalized tables exist. |
| Observability of persistence | Add metrics: snapshot duration, bytes written, entity counts. |

## Alternatives Considered
1. **Immediate normalized relational schema**: Rejected for higher upfront design and migration complexity without empirical workload data.
2. **Event sourcing (append-only log)**: Deferred; would add complexity to query paths and rule evaluation prematurely.
3. **BoltDB / Badger**: SQLite chosen for ubiquity, ecosystem maturity, and SQL inspection/debuggability.

## Implementation Notes
- The concrete implementation type is `sqlite.Store` in `internal/infra/persistence/sqlite` (migrated from the deprecated `internal/persistence/sqlite` path); the public helper `core.NewSQLiteStore` returns that type while preserving the historical constructor name for backwards compatibility and readability in calling code.
- Env vars: `COLONYCORE_STORAGE_DRIVER`, `COLONYCORE_SQLITE_PATH`, `COLONYCORE_POSTGRES_DSN`.
- Test coverage via `internal/core/sqlite_store_test.go` and `internal/infra/persistence/sqlite/store_test.go` ensures snapshot reload semantics.

## Driver Selection & Configuration
Persistent storage drivers are selected exclusively via environment variables—deployments can move between in-memory, SQLite, or Postgres (reserved) without recompilation.

### Quick Reference
- Unset `COLONYCORE_STORAGE_DRIVER` → `sqlite` (creates `./colonycore.db`; override path with `COLONYCORE_SQLITE_PATH`).
- `COLONYCORE_STORAGE_DRIVER=memory` → in-memory / ephemeral store.
- `COLONYCORE_STORAGE_DRIVER=sqlite` → explicit SQLite selection (same defaults as unset).
- `COLONYCORE_STORAGE_DRIVER=postgres` → reserved placeholder; requires `COLONYCORE_POSTGRES_DSN`.

### Environment Variables
- `COLONYCORE_STORAGE_DRIVER`: `memory` | `sqlite` | `postgres`. Defaults to `sqlite` outside of tests.
- `COLONYCORE_SQLITE_PATH`: Filesystem path to the SQLite DB file (`./colonycore.db` if unset).
- `COLONYCORE_POSTGRES_DSN`: Connection string for the upcoming Postgres implementation.

Local development and CI can rely on defaults (SQLite file under the current working directory). Tests still target the in-memory driver directly for performance.

## Status & Follow-Up
Accepted for v0.1.0 baseline. Postgres driver implementation tracked as a follow-up (will supersede placeholder). Metrics & delta persistence targeted post initial adopter feedback.

## SQLite Operational Notes
The snapshot strategy writes the full logical state into a single `state` table as JSON blobs after each successful transaction. This keeps the implementation compact, allows readers to inspect state with standard SQL tooling, and makes rollback/reset trivial (delete the DB file). The trade-offs are:
- Write throughput remains serialized; concurrent writes queue behind the transaction boundary.
- Snapshot cost grows with dataset size; future optimizations focus on incremental/delta persistence and background checkpointing.

Read concurrency is comparable to the in-memory driver (multiple readers, single writer). For larger deployments, Postgres will deliver row-level locking and normalized schemas.

## Postgres Roadmap
The placeholder driver checks configuration wiring today so deployments can standardize on `COLONYCORE_POSTGRES_DSN` early. Planned work includes:
1. Normalized schema with per-entity persistence.
2. Row-level locking and transactional alignment with the rules engine.
3. Versioned migrations and schema introspection.
4. Metrics covering snapshot lag, row counts, and write latency.

Once complete, the Postgres driver will replace the placeholder implementation without changing the configuration contract.
