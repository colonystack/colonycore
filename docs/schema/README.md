# Schema Index

This directory holds machine-readable contracts. `entity-model.json` is the seed for Entity Model v0 per ADR-0003 and will drive generated Go types, OpenAPI, DDL, and fixtures in later slices.

Conventions:
- IDs are opaque UUIDv7 strings for all entities.
- `id`, `created_at`, and `updated_at` are required on every entity.
- Enums capture lifecycle/status sets; `states.enum` references the enum name declared under `enums`. Housing lifecycle uses `housing_state` (quarantine → active → cleaning → decommissioned), and protocol/permit compliance states follow RFC-0001 §5.3.
- Natural keys document uniqueness scopes (global, facility, authority, line, etc.) but primary keys remain opaque IDs.
- Extension slots (`attributes`, `environment_baselines`, `pairing_attributes`, etc.) are plugin-safe maps; schema-specific extensions belong in plugins, not core.

Validation:
- Run `make entity-model-validate` (also wired into pre-commit) to sanity-check the JSON (presence of enums, required fields, and relationship targets). The target only reads `docs/schema/entity-model.json` and keeps domain layering intact.
