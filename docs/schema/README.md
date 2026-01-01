# Schema Index

This directory holds machine-readable contracts. `entity-model.json` is the seed for Entity Model v0 per ADR-0003 and drives generated Go types, OpenAPI, DDL, fixtures, and ERDs.

## Governance registry schema

`docs/schema/registry.schema.json` defines the JSON Schema for `docs/rfc/registry.yaml`. The `cmd/registry-check` CLI loads this schema during `make registry-lint`, and registry fixtures live under `testutil/fixtures/registry`.

Conventions:
- IDs are opaque UUIDv7 strings for all entities.
- `id`, `created_at`, and `updated_at` are required on every entity.
- Enums capture lifecycle/status sets; `states.enum` references the enum name declared under `enums`. Housing lifecycle uses `housing_state` (quarantine → active → cleaning → decommissioned), and protocol/permit compliance states follow RFC-0001 §5.3. Enum values must be non-empty and deduplicated.
- Natural keys document uniqueness scopes (global, facility, authority, line, etc.) but primary keys remain opaque IDs.
- Relationship cardinalities use `0..1`, `1..1`, `0..n`, or `1..n` notation only; the validator rejects other forms to keep generators aligned.
- Properties must declare either a type or `$ref` so downstream generators can map them into code, OpenAPI, and DDL.
- Extension slots (`attributes`, `environment_baselines`, `pairing_attributes`, etc.) are plugin-safe maps; schema-specific extensions belong in plugins, not core.
- Dataset and plugin facades normalize lifecycle/status/environment values to the generated enums before exposing them to plugins; unknown inputs fall back to canonical defaults (planned/scheduled/stored/active/terrestrial) to avoid drifting from `entity-model.json`.

Validation & targets:
- Run `make entity-model-verify` (also executed by `make lint`) to sanity-check the JSON: semver version, required base fields, relationship cardinalities/targets, non-empty enums, allowlisted invariants, property enum references, and type/$ref presence. This target keeps domain layering intact by only reading `docs/schema/entity-model.json`.
- `make entity-model-generate` emits:
  - Go enums and struct projections into `pkg/domain/entitymodel`.
  - OpenAPI components to `docs/schema/openapi/entity-model.yaml`.
  - Postgres/SQLite DDL to `docs/schema/sql/{postgres.sql,sqlite.sql}`.
  - ERD assets to `docs/annex/entity-model-erd.{dot,svg}`.
  - Canonical fixtures to `testutil/fixtures/entity-model/snapshot.json` used by invariant conformance tests.
  `make entity-model-verify` runs validation and generation together.
- The generated OpenAPI components are embedded for runtime use via `internal/entitymodel.OpenAPISpec`/`NewOpenAPIHandler` so handlers and clients can serve the canonical contract without shelling out to the generator.
- Drift guards:
  - `make lint`/`make entity-model-generate` will rewrite all generated artifacts (including fixtures) from `entity-model.json`.
  - `internal/tools/entitymodel/generate/main_test.go` fails if committed outputs drift from the generator (Go code and OpenAPI), forcing contributors to update artifacts alongside schema edits.
  - `internal/core/rules_invariants_test.go` keeps the schema-declared invariants in lockstep with the default rule set so enforcement cannot lag the contract.
- For a human-readable entry point that links the canonical assets without duplicating the schema, see `docs/annex/entity-model-overview.md`.
