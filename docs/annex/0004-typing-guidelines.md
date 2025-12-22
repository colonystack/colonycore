# Annex 0004: Typing Guidelines (any usage)

- Status: Draft
- Linked RFCs: 0001-colonycore-base-module
- Owners: Core Maintainers
- Last Updated: 2025-12-22

## Purpose
Use `any` only at explicit untyped boundaries to keep API contracts clear and refactor-safe.
This policy applies across the repo and is especially strict for public surfaces in
`pkg/pluginapi` and `pkg/datasetapi` (see ADR-0009).

## Allowed uses of `any`
- JSON/codec boundaries where the payload shape is intentionally open, such as
  `map[string]any`, `[]map[string]any`, or `map[string]map[string]any`.
- Third-party API shims when the upstream library returns `any`; wrap and convert to
  typed structures as early as possible.
- Reflection utilities that must operate on arbitrary values and keep `any` contained.
- Unconstrained generics using `T any` in type parameter lists.
- Tests may use `any` for fixtures or assertions, but should not leak into production
  types or public APIs.

## Disallowed uses of `any`
- Public API types or method signatures, except the explicit JSON/codec boundaries above.
- Domain model types or core service contracts that can be expressed as concrete types.
- Using `any` to avoid modeling a known data shape.

## Preferred alternatives
- Concrete structs or narrowly scoped interfaces.
- `json.RawMessage` when a raw JSON payload must be passed through.
- Typed payload wrappers that keep untyped maps at a single boundary.

## Exceptions and guard allowlist
Any exception must be explicit, documented, and limited:
- Record the exception in the guard allowlist (file path + rationale + owner).
- Prefer file-level or symbol-level exceptions; avoid line-level unless unavoidable.
- Update this policy and the allowlist together to keep review transparent.

## Layering and contract constraints
- Follow dependency direction in `ARCHITECTURE.md`; do not introduce imports that
  violate layer boundaries or import-boss rules.
- When public APIs change, update the snapshots under `internal/ci/` per ADR-0009.
- Keep entity-model derived types aligned with ADR-0003; avoid manual drift.

## Trade-offs
- Tighter typing improves refactoring safety but can reduce extensibility. Prefer small
  stable interfaces and JSON/codec boundaries for extension data.
- Conversions at boundaries can add allocations; measure hot paths and document waivers
  if needed.

## References
- ARCHITECTURE.md
- docs/adr/0003-core-domain-schema.md
- docs/adr/0009-plugin-interface-stability-and-versioning.md
- docs/adr/0010-contextual-accessor-pattern.md
- CONTRIBUTING.md
