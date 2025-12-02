# Schema Index

This directory holds machine-readable contracts. `entity-model.json` is the seed for Entity Model v0 per ADR-0003 and will drive generated Go types, OpenAPI, DDL, and fixtures in later slices.

Conventions:
- IDs are opaque UUIDv7 strings for all entities.
- `id`, `created_at`, and `updated_at` are required on every entity.
- Enums capture lifecycle/status sets; `states.enum` references the enum name declared under `enums`. Housing lifecycle uses `housing_state` (quarantine → active → cleaning → decommissioned), and protocol/permit compliance states follow RFC-0001 §5.3. Enum values must be non-empty and deduplicated.
- Natural keys document uniqueness scopes (global, facility, authority, line, etc.) but primary keys remain opaque IDs.
- Properties must declare either a type or `$ref` so downstream generators can map them into code, OpenAPI, and DDL.
- Extension slots (`attributes`, `environment_baselines`, `pairing_attributes`, etc.) are plugin-safe maps; schema-specific extensions belong in plugins, not core.

Validation & targets:
- Run `make entity-model-verify` (also executed by `make lint`) to sanity-check the JSON: semver version, required base fields, relationship cardinalities/targets, non-empty enums, allowlisted invariants, property enum references, and type/$ref presence. This target keeps domain layering intact by only reading `docs/schema/entity-model.json`.
- `make entity-model-generate` emits Go enums and struct projections into `pkg/domain/entitymodel` via `internal/tools/entitymodel/generate`; extend this target with OpenAPI/DDL/ERD generation as those pieces land. `make entity-model-verify` runs validation and generation together.
