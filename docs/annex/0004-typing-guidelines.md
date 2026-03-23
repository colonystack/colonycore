# Annex 0004: Typing Guidelines (any usage)

- Status: Accepted
- Linked RFCs: 0001-colonycore-base-module
- Owners: Core Maintainers
- Last Updated: 2025-12-29

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
- `ObjectPayload`/`ExtensionPayload` for extension payload access (`ExtensionSet.Get/Core`).
- `ChangePayload` for rule change snapshots (`Change.Before/After`).

### Dataset parameter defaults

For `pkg/datasetapi.Parameter` examples/defaults, prefer `json.RawMessage` to keep
the public surface free of `any` while preserving JSON flexibility. Decode and
validate against `Parameter.Type` at the boundary so defaults follow the same
coercion rules as supplied values.
Host templates now decode JSON defaults and apply the same coercion rules as
runtime-supplied values.

### Change snapshots

Rule change snapshots are exposed via `ChangePayload` and carry JSON bytes that
should be decoded at the boundary (for example into `map[string]any` or a typed
struct). Domain `Change.Before`/`Change.After` use `ChangePayload`, and core
rules should decode per entity type before evaluation.

## Exceptions and guard allowlist

Any exception must be explicit, documented, and limited:

- Record the exception in the guard allowlist as an exact `anyguard` selector.
- Keep the description tied to the JSON/codec boundary or helper that still
  requires `any`.
- Update this policy and the allowlist together to keep review transparent.

### Allowlist format and location

Store the allowlist in `internal/ci/any_allowlist.yaml` to keep it close to API
snapshots and outside layered packages.

Format (YAML, `anyguard` schema version `2`):

- Top-level:
  - `version`: integer schema version (`2`).
  - `exclude_globs`: list of globs to ignore (for example `**/*_test.go`).
  - `entries`: array of allowlist entries.
- Entry fields:
  - `selector.path`: repo-relative file path.
  - `selector.owner`: owning type or function name reported by `anyguard`.
  - `selector.category`: exact AST child slot reported by `anyguard`
    (for example `*ast.MapType.Value`).
  - `selector.line` / `selector.column`: exact coordinates for the finding.
  - `description`: short explanation tied to this policy.
  - `refs`: optional list of relevant docs (ADR/RFC/annex paths).

Guard behavior:

- Match the canonical finding identity exactly:
  `{path, owner, category, line, column}`.
- Keep entries fully resolved; this repository does not rely on legacy
  selector matching without coordinates.
- Exclude tests via `exclude_globs` rather than per-entry.
- Stale or ambiguous selectors fail closed and must be updated together with the
  source change.

## Guard implementation

The lint-time guard runs through `anyguard` v2.0.2 as a golangci-lint module
plugin. `make golangci` builds the custom binary from `.custom-gcl.yml` and
executes it with the repository's `.golangci.yml`. For local verification:

```bash
make golangci
```

Default roots enforced by the guard:

- `pkg/pluginapi`
- `pkg/datasetapi`
- `pkg/domain`
- `internal/core`
- `internal/adapters/datasets`
- `internal/infra/blob`
- `internal/infra/persistence`
- `plugins`

Override the defaults by editing the `linters.settings.custom.anyguard.settings.roots`
list in `.golangci.yml` when the audit scope changes.

## Layering and contract constraints

- Follow dependency direction in `ARCHITECTURE.md`; do not introduce imports that
  violate layer boundaries or import-boss rules.
- When public APIs change, update the snapshots under `internal/ci/` per ADR-0009.
- Keep entity-model-derived types aligned with ADR-0003; avoid manual drift.

## Touchpoint scope (typing hardening audit)

The audit scope matches the guard roots and `.import-restrictions` for:
`pkg/domain`, `internal/core`, `internal/adapters/datasets`,
`internal/infra/blob`, `internal/infra/persistence`,
`pkg/pluginapi`, `pkg/datasetapi`, and `plugins/*`. Validate dependency
direction against `ARCHITECTURE.md` before changing imports.

## Code-owner review checklist

- `make lint` passes, including the `anyguard` module plugin and import-boss
  guards.
- Any `any` usage is limited to JSON/codec boundaries or documented allowlist
  entries in `internal/ci/any_allowlist.yaml`.
- Public API changes (if any) update `internal/ci/{pluginapi,datasetapi}.snapshot`
  and keep `pkg/pluginapi`/`pkg/datasetapi` free of `internal/**` or `pkg/domain`
  imports.
- No manual edits to ADR-0003 generated artifacts (use `make entity-model-*`).
- Dependency direction remains aligned with `ARCHITECTURE.md` across touched
  layers.

## Trade-offs

- Tighter typing improves refactoring safety but can reduce extensibility. Prefer small
  stable interfaces and JSON/codec boundaries for extension data.
- Conversions at boundaries can add allocations; measure hot paths and document waivers
  if needed.

## Benchmark tracking

Baseline microbenchmarks cover JSON-boundary clone paths so regressions stay visible.

- `pkg/datasetapi`: `BenchmarkDeepCloneAttributes`, `BenchmarkExtensionPayloadMap`
- `pkg/pluginapi`: `BenchmarkCloneValueNested`, `BenchmarkObjectPayloadMap`, `BenchmarkExtensionSetRaw`
- Baseline snapshot: `internal/ci/benchmarks/baseline.withmeta.results`
- Runner: `scripts/benchmarks/ci.sh` (Sweet → aggregated results → benchstat)

## Changelog

### Added

- `anyguard` module-plugin enforcement with an exact-selector allowlist
  (`internal/ci/any_allowlist.yaml`).

### Changed

- Public API surfaces restrict `any` to explicit JSON/codec boundaries per this annex.
- `pkg/pluginapi.Change` before/after snapshots now use `ChangePayload` (`json.RawMessage`).
- `pkg/datasetapi.Parameter` `Example` and `Default` values are now `json.RawMessage`.

### Migration notes

- Plugins and dataset templates must decode `ChangePayload.Raw()` and parameter `Example`/`Default` as JSON bytes.
- Extension payload access should use `ObjectPayload`/`ExtensionPayload`
  wrappers and contextual helpers; raw maps remain JSON-boundary only.
- Allowlist maintenance now follows the `anyguard` selector model; changes that
  move or split `any` usage must refresh the exact selector coordinates.

## References

- ARCHITECTURE.md
- docs/adr/0003-core-domain-schema.md
- docs/adr/0009-plugin-interface-stability-and-versioning.md
- docs/adr/0010-contextual-accessor-pattern.md
- CONTRIBUTING.md
