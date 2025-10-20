# ColonyCore

ColonyCore is an extensible base module for laboratory colony management. It couples a rules-driven core domain service with pluggable species modules and tooling to validate the project registry.

## Project layout
- `pkg/domain/` contains the pure domain model (entities, value objects, rule contracts) without infrastructure dependencies.
- `internal/core/` contains application services, orchestration logic, rules engine integration, and use-case level coordination.
- `internal/infra/persistence/sqlite/` houses the in-memory transactional store and SQLite snapshot-backed persistent implementation (migrated from the legacy `internal/persistence/sqlite/` path now removed).
- `plugins/` hosts externally consumable plugins (for example `plugins/frog`) that register species-specific schemas and rules.
- `cmd/registry-check/` provides the CLI used to validate `docs/rfc/registry.yaml` against the expected structure.
- `docs/` captures design history (`docs/adr/`), planning RFCs (`docs/rfc/`), operational annexes (`docs/annex/`), and machine-readable schemas (`docs/schema/`).
- `Makefile` orchestrates common build, lint, and test workflows.

## Getting started
1. Install [Go](https://go.dev/dl/) 1.25 or newer and ensure `$GOPATH/bin` is on your `PATH`.
2. Clone this repository and change into its directory.
3. Run `go test ./...` (or `make test`) to verify your toolchain and dependencies.

## Development workflow
- Build all packages with `make build`.
- Compile the registry validator via `make registry-check`, which outputs `cmd/registry-check/registry-check`.
- Validate the governance registry using `make registry-lint` or by running `go run ./cmd/registry-check --registry docs/rfc/registry.yaml`.
- Refer to `CONTRIBUTING.md` for coding standards, workflow expectations, and pull request guidance.

### Storage

Local development defaults to an embedded SQLite database file (`colonycore.db`) created in the current working directory, providing durable state across restarts. Unit tests continue to use the in-memory store directly. Set `COLONYCORE_STORAGE_DRIVER` to `memory`, `sqlite`, or `postgres` (placeholder) to select a driver. The blob/object store also works out of the box: if unset, `COLONYCORE_BLOB_DRIVER=fs` with its root at `./blobdata` (created automatically). See ADR-0007 (`docs/adr/0007-storage-baseline.md`) and ADR-0008 (`docs/adr/0008-object-storage-contract.md`) for detailed configuration, rationale, and roadmap (including blob driver options and S3 configuration variables).

**At a glance:** You can switch between all supported persistence and blob drivers purely by setting environment variables—no code changes or recompilation are required, and safe local defaults are used when variables are unset.

Cheat sheet:

Persistent store:
* (default) unset: SQLite file `./colonycore.db` (override path with `COLONYCORE_SQLITE_PATH`)
* `COLONYCORE_STORAGE_DRIVER=memory`: ephemeral, non-durable
* `COLONYCORE_STORAGE_DRIVER=sqlite`: explicit sqlite (optionally set `COLONYCORE_SQLITE_PATH`)
* `COLONYCORE_STORAGE_DRIVER=postgres`: requires `COLONYCORE_POSTGRES_DSN`

Blob / object store:
* (default) unset or `COLONYCORE_BLOB_DRIVER=fs`: filesystem at `./blobdata` (override root with `COLONYCORE_BLOB_FS_ROOT`)
* `COLONYCORE_BLOB_DRIVER=s3`: S3 or S3-compatible (requires `COLONYCORE_BLOB_S3_BUCKET`; optional `COLONYCORE_BLOB_S3_REGION`, `COLONYCORE_BLOB_S3_ENDPOINT` & `COLONYCORE_BLOB_S3_PATH_STYLE=true` for MinIO)
* `COLONYCORE_BLOB_DRIVER=memory`: ephemeral, non-durable (tests)

Combined example (Postgres + MinIO):

```bash
export COLONYCORE_STORAGE_DRIVER=postgres
export COLONYCORE_POSTGRES_DSN='postgres://colonycore:colonycore@localhost:5432/colonycore?sslmode=disable'

export COLONYCORE_BLOB_DRIVER=s3
export COLONYCORE_BLOB_S3_BUCKET=colonycore-dev
export COLONYCORE_BLOB_S3_REGION=us-east-1
export COLONYCORE_BLOB_S3_ENDPOINT=http://localhost:9000
export COLONYCORE_BLOB_S3_PATH_STYLE=true
```

### Optional Postgres (Experimental)

You do **not** need any external services (containers, databases, object stores) for normal local development—the default embedded SQLite + filesystem blob store work out of the box. A `docker-compose.yml` is included only to spin up a Postgres 16 instance for experimenting with the future high-concurrency driver. The Postgres driver is still a placeholder; behavior and schema are subject to change.

To try Postgres locally (purely optional):

```bash
docker compose up -d            # start the postgres container
export COLONYCORE_STORAGE_DRIVER=postgres
export COLONYCORE_POSTGRES_DSN='postgres://colonycore:colonycore@localhost:5432/colonycore?sslmode=disable'

# Run tests or your binary as usual; the placeholder driver may not yet implement all features.
```

Tear down when finished:

```bash
docker compose down -v
```

If the environment variable `COLONYCORE_STORAGE_DRIVER` is unset, the code will ignore the running Postgres container and continue using the embedded SQLite store.

## Dataset analytics
- The dataset REST surface is documented in `docs/schema/dataset-service.openapi.yaml` and exposes
  template enumeration, parameter validation, streaming results (JSON/CSV), and asynchronous exports.
- Species plugins register versioned dataset templates through the core plugin registry; see
  `internal/core/dataset.go` and the frog reference module (`plugins/frog`) for a working example aligned
  with RFC 0001's reporting guidance.
- A background export worker (package `internal/adapters/datasets`) applies RBAC scope filters, emits audit entries,
  stores signed artifacts in the managed object store, and serves export status via `/api/v1/datasets/exports`.
- Sample analyst clients are provided in `clients/python/dataset_client.py` (requests-based) and
  `clients/R/dataset_client.R` (httr-based) to illustrate reproducing exports from external runtimes.

## Plugin Development
ColonyCore enforces hexagonal architecture through comprehensive contextual accessor patterns:

### Core Principles
- **No Raw Constants**: Access domain values via context providers, never raw constants
- **Contextual Interfaces**: Use `pluginapi.NewEntityContext()`, `NewActionContext()`, `NewSeverityContext()`, `NewLifecycleStageContext()`, `NewHousingContext()`, `NewProtocolContext()`
- **Semantic References**: All contexts return opaque reference types with semantic methods
- **Builder Patterns**: Create violations via fluent builders: `NewViolationBuilder().WithEntityRef(entityContext.Organism()).BuildWarning()`

### Updated Patterns (v0.2.0)
- **Environment Contexts**: Use `housingContext.Aquatic().IsAquatic()` instead of string comparisons
- **Status Contexts**: Use `protocolContext.Active().IsActive()` instead of raw status checks  
- **Facade Contextual Methods**: Call `organism.GetCurrentStage().IsActive()` for lifecycle queries
- **Provider Interfaces**: Access formats via `datasetapi.GetFormatProvider().JSON()` instead of constants

### Example Usage
```go
// Contextual environment checking
housingCtx := pluginapi.NewHousingContext()
if housing.GetEnvironmentType().Equals(housingCtx.Aquatic()) {
    // Handle aquatic environment
}

// Contextual lifecycle checking  
if organism.IsActive() && !organism.IsRetired() {
    // Process active organism
}

// Provider-based format access
formatProvider := datasetapi.GetFormatProvider()
template.OutputFormats = []datasetapi.Format{
    formatProvider.JSON(),
    formatProvider.CSV(),
}
```

### Migration Guide
The contextual accessor pattern is enforced via:
- **Compile-time**: Provider interfaces replace raw constants
- **Runtime Tests**: Integration tests validate pattern compliance
- **AST Analysis**: Anti-pattern detection scans for forbidden raw constant usage

See the `plugins/frog` reference implementation for complete patterns and ADR-0009 for stability guarantees.

## Testing
- `make test` runs the race-enabled Go test suite and enforces the configured coverage threshold.
- `make lint` chains formatting checks, vetting, registry validation, and `golangci-lint` (install instructions are in the Makefile). Use `SKIP_GOLANGCI=1 make lint` to bypass the aggregated linter when necessary.
- Additional manual validation steps are described in relevant documents under `docs/`.

### Integration smoke test
The repository includes a deliberately tiny end-to-end integration smoke test at `internal/integration/smoke_test.go`. Its purpose is fast CI health coverage across all in-process adapters without introducing a large maintenance surface.

What it exercises per adapter:
* Persistent store variants (memory + sqlite): creates one housing unit and one organism, assigns the organism to the housing unit (thereby touching validation + assignment logic), and verifies the data is persisted via list/get calls.
* Blob store variants (memory, filesystem, mock S3): writes a single blob (text payload), reads it back to verify content, and deletes it.

This is intentionally more than a single-record round trip for the core store so that assignment logic and rule evaluation are covered, but it stays minimal (two records + one relation) to keep runtime negligible.

Run just the smoke test:
```bash
go test -run TestIntegrationSmoke ./internal/integration -count=1
```

## Documentation and support
- Review `CODE_OF_CONDUCT.md` and `CONTRIBUTING.md` before opening issues or pull requests.
- Ownership and escalation paths are tracked in `MAINTAINERS.md` and `SECURITY.md`; the latter also details the coordinated vulnerability disclosure process.
- Long-form decision records and proposals live in `docs/adr/` and `docs/rfc/`. Operational considerations are captured in `docs/annex/`.
- All source code is licensed under the terms of `LICENSE`.
- Non-human contributors must follow the workflows and guardrails in `AGENTS.md`.
