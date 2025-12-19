# ADR 0003: Core Domain Schema Normalization

- Status: Accepted
- Deciders: Tobias Harnickell
- Date: 2025-09-21
- Related RFCs: 0001-colonycore-base-module

## Context
Define canonical relational structures, temporal history strategy, and JSON extension patterns for ColonyCore entities.

## Decision
ColonyCore adopts a single canonical entity contract named **Entity Model v0**. The contract is stored as versioned JSON Schema at `docs/schema/entity-model.json` and drives every generated artifact that represents core entities. The model is authoritative for field definitions, identifier semantics, lifecycle states, and cross-entity constraints. Code and documentation MUST derive from this source; manual drift is forbidden.

### Canonical Data Contract
- Every core entity listed in RFC-0001 (Organism, Cohort, BreedingUnit, HousingUnit, Facility, Procedure, Treatment, Observation, Sample, Line, Strain, Protocol, Project, Permit, SupplyItem) is represented in `entity-model.json`.
- Each entity definition declares:
  - `id`: opaque, globally unique identifier (UUIDv7) used as the persistent primary key.
  - Optional `naturalKeys`: stable domain identifiers (e.g., facility code, protocol accession) with uniqueness scopes.
  - `requiredFields`: species-agnostic attributes, their JSON types, nullability, and validation constraints.
  - `relationships`: outbound foreign keys with cardinalities (1:1, 1:n, n:m) and referential actions.
  - `states`: lifecycle state machines with allowed transitions and terminal markers.
  - `invariants`: rule expressions (capacity checks, lineage constraints, protocol coverage) referenced by name for enforcement.
- Relationships are defined using canonical entity references; plugin-specific extensions MUST attach through the documented extension hooks instead of modifying core fields.

### Generated Artifacts
- Go structs in `pkg/domain/entitymodel` are generated from `entity-model.json` and include struct tags for JSON, database columns, and OpenAPI binding.
- Database DDL for Postgres (normalized schema) and SQLite (mapping layer) is generated under `docs/schema/sql/`.
- OpenAPI components for create, update, and read DTOs are generated under `docs/schema/openapi/entity-model.yaml` and consumed by API handlers.
- Plugin contract documentation (`docs/annex/plugin-contract.md`) and the static analyzer fed by `scripts/validate_plugin_contract.go` are generated from the same model to enumerate mandatory fields and permitted extension points.
- An ERD diagram (`docs/annex/entity-model-erd.svg`) is rendered from the model using `make entity-model-erd`.
- Synthetic datasets and fixtures (`testutil/fixtures/entity-model/*.json`) are produced from the model to exercise constraints in CI.

### Tooling & Usage
- Validate the logical schema: `make entity-model-validate`.
- Regenerate code, OpenAPI, and DDL: `make entity-model-generate` (runs as part of `make lint`).
- Full verification (validate + generate): `make entity-model-verify`.
- Render the ERD: `make entity-model-erd` (requires Docker; spins up a temporary Postgres container, loads the generated Postgres DDL with `psql -X -v ON_ERROR_STOP=1 -1`, runs SchemaSpy, and writes `docs/annex/entity-model-erd.svg` plus `docs/annex/entity-model-erd.dot`; the full HTML report lives under `.cache/schemaspy/entitymodel-erd`).

### Change Control
- `entity-model.json` is versioned using SemVer (`version` field) and follows ADR-0009 for compatibility. Breaking schema changes require an RFC and MAJOR bump; additive fields follow MINOR.
- Changes to the model must pass the schema diff validator (`make entity-model-diff`), which reports breaking removals, invariant rewrites, or transition alterations.
- Pull requests touching the model MUST update generated artifacts by running `make entity-model-generate` and commit outputs together.
- Plugin authors receive compatibility guidance via release notes generated from the diff (enum changes, new invariants). Core code may not rely on plugin-provided fields for invariants unless encoded in the base model.

### Governance and Enforcement
- CI runs `make entity-model-verify`, which executes:
  1. JSON Schema validation for `entity-model.json`.
  2. Round-trip generation tests (model → Go/OpenAPI/DDL → model) to detect drift.
  3. Synthetic dataset load into SQLite/Postgres adapters with referential integrity checks and lifecycle transition coverage (≥95% pass threshold).
  4. Static analysis ensuring plugins only extend via approved hooks and that core packages remain species-agnostic.
- Feature development that impacts the entity model requires aligning RFC updates before merging code. Unexpected divergences between code and model fail CI and block the PR.

## Consequences
### Positive
- Single source of truth eliminates schema drift across storage adapters, APIs, and documentation.
- Generated artifacts accelerate adoption of Postgres normalization while keeping SQLite parity.
- Static enforcement guards prevent species-specific leakage into core packages, upholding ADR-0009 guarantees.
- Synthetic datasets improve confidence in lifecycle invariants and migration readiness.

### Negative
- Initial investment in the generation pipeline and fixtures increases complexity for contributors.
- Strict change control can slow experimentation; feature spikes must work through the model diff workflow.
- Large model diffs may be harder to review without tooling support; reviewers depend heavily on generated change reports.

### Mitigations
- Provide contributor documentation and `make` targets to guide model edits and artifact regeneration.
- Offer preview tooling (`make entity-model-report`) that renders HTML summaries, easing review.
- Maintain backward-compatible adapters and migration scripts to smooth Postgres rollout.
