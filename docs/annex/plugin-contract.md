# Plugin Contract (Entity Model v0)

Status: Outline — will be generated from `docs/schema/entity-model.json` per ADR-0003 and ADR-0009.

Scope:
- Defines the species-agnostic fields every plugin must respect for core entities (IDs, timestamps, lifecycle enums, natural keys, invariants such as `housing_capacity`, `lineage_integrity`, `protocol_coverage`, `protocol_subject_cap`, and `lifecycle_transition`).
- Enumerates approved extension hooks (`attributes`, `environment_baselines`, `pairing_attributes`, etc.) and forbids plugin-specific fields from being injected into core entity definitions.
- Documents lifecycle/state enums and referential rules so plugins can validate input without copying constants.

Process:
- The canonical contract will be generated alongside Go/OpenAPI/DDL artifacts; until then, this outline anchors the doc path for CI/static-analysis wiring.
- Plugin authors must follow the contextual accessor pattern from ADR-0010 and the stability guarantees in ADR-0009; raw core structs/constants remain off-limits.
