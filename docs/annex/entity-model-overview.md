# Entity Model Overview (v0)

Purpose: provide a human-readable entry point into the canonical entity contract without duplicating the JSON Schema. The source of truth remains `docs/schema/entity-model.json` (ADR-0003, RFC-0001).

## Canonical artifacts
- Schema: `docs/schema/entity-model.json` (versioned, semver; fingerprinted by `make entity-model-diff`)
- OpenAPI: `docs/schema/openapi/entity-model.yaml` (served via `internal/entitymodel.NewOpenAPIHandler`)
- DDL: `docs/schema/sql/{postgres.sql,sqlite.sql}` and runtime helpers in `internal/entitymodel/sqlbundle`
- ERD: `docs/annex/entity-model-erd.svg` (`.dot` alongside)
- Plugin contract: `docs/annex/plugin-contract.md` (generated from the schema; enforced by `scripts/validate_plugin_patterns.go`)
- Fixtures: `testutil/fixtures/entity-model/snapshot.json` (load-tested across memory/sqlite/postgres)

## Entities
Covered per RFC-0001: Organism, Cohort, BreedingUnit, HousingUnit, Facility, Procedure, Treatment, Observation, Sample, Line, Strain, Protocol, Project, Permit, SupplyItem, GenotypeMarker. Each embeds `id`, `created_at`, `updated_at` and uses the schema’s required/optional fields, natural keys, relationships, and enums.

Lifecycle/status enums are defined once in the schema and exported through generated Go/Plugin/ Dataset API constants. Invariants are schema-bound and mapped to rules: `housing_capacity`, `protocol_subject_cap`, `lineage_integrity`, `lifecycle_transition`, `protocol_coverage`.

## How to consume
- Validate/generate: `make entity-model-verify` (runs from `make lint`), `make entity-model-diff` to check the fingerprint.
- Serve OpenAPI: wire `internal/entitymodel.NewOpenAPIHandler` into admin/debug endpoints.
- Apply storage schema: use `internal/entitymodel/sqlbundle.{SQLite,Postgres}` with `SplitStatements` in adapters; Postgres/SQLite/memory parity is exercised via fixtures and rules tests.
- Extensibility: plugins must stick to the mandatory fields and extension hooks listed in `docs/annex/plugin-contract.md`; static checks run from `scripts/validate_plugin_patterns.go`.

## Change control
- Schema edits must bump `version`, regenerate artifacts (`make entity-model-verify`), and keep diffs in sync with ADR-0003 expectations.
- Cardinalities are limited to `0..1`, `1..1`, `0..n`, `1..n`; required arrays carry `minItems` and are enforced consistently across adapters.
- `facility.housing_unit_ids` is `derived` by design to avoid denormalizing the FK stored on `housing_units.facility_id`.
