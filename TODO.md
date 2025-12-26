# TODO: Typing Hardening (any usage)

## References (read/align)
- ARCHITECTURE.md
- docs/adr/0003-core-domain-schema.md
- docs/adr/0009-plugin-interface-stability-and-versioning.md
- docs/adr/0010-contextual-accessor-pattern.md
- docs/annex/0004-typing-guidelines.md
- CONTRIBUTING.md
- docs/annex/0003-import-boss-runbook.md

## Scope + boundaries
- [x] Confirm layers/packages in scope (expected: pkg/pluginapi, pkg/datasetapi, docs/).
- [x] Confirm no cross-layer import changes; update .import-restrictions only if new deps are required.
- [x] Identify generated artifacts that may need refresh (internal/ci/pluginapi.snapshot, internal/ci/datasetapi.snapshot, entity-model outputs if touched).
- [x] If Change payloads are narrowed, confirm internal/core adapter updates are in scope and keep import direction intact.

## Policy
- [x] Create docs/annex/0004-typing-guidelines.md with allowed any usage and an exception/whitelist mechanism.
- [x] Record trade-offs (tight interfaces vs extensibility) and note compatibility posture per ADR-0009.

## Inventory + classification
  - [x] Enumerate all public any usage in pkg/pluginapi and pkg/datasetapi.
  - [x] Classify each use as allowed boundary vs disallowed; capture a whitelist of allowed locations for guard tooling.
  - [x] pkg/pluginapi/views.go: Attributes/CoreAttributes/EnvironmentBaselines/Data/ChainOfCustody return map[string]any or []map[string]any - allowed boundary (JSON/extension payload).
  - [x] pkg/pluginapi/extensions.go: ExtensionSet Get/Core return ObjectPayload; Raw map is map[string]map[string]map[string]any JSON boundary.
  - [x] pkg/pluginapi/domain_aliases.go: Change/ChangeBuilder Before/After now use ChangePayload (json.RawMessage snapshot boundary).
  - [x] pkg/pluginapi/plugin.go: RegisterSchema schema map[string]any - allowed boundary (JSON schema).
  - [x] pkg/pluginapi/payload.go: ObjectPayload map[string]any - allowed boundary (JSON payload).
  - [x] pkg/datasetapi/types.go: Parameter Example/Default any - disallowed; replace with json.RawMessage. RunRequest Parameters map[string]any; Row map[string]any; RunResult Metadata map[string]any; TemplateRuntime ValidateParameters/Run map[string]any - allowed boundary (JSON/codec).
  - [x] pkg/datasetapi/extensions.go: ExtensionSet Get/Core return ExtensionPayload; Raw map is map[string]map[string]map[string]any JSON boundary.
  - [x] pkg/datasetapi/payload.go: ExtensionPayload map[string]any - allowed boundary (JSON payload).
  - [x] pkg/datasetapi/facade.go: Attributes/CoreAttributes/EnvironmentBaselines/PairingAttributes/Data/ChainOfCustody serialization uses map[string]any or []map[string]any - allowed boundary (extension payload).
  - [x] pkg/datasetapi/host_template.go: parameter helpers return any internally - exception needed (internal helper); guard should exclude via symbol or file allowlist.
  - [x] Confirm *_test.go any usage stays test-only or is explicitly excluded from guard scope (allowed by policy).

### Guard allowlist (implemented in internal/ci/any_allowlist.json)
- pkg/pluginapi/views.go: JSON boundary for view attributes/custody payloads.
- pkg/pluginapi/payload.go: ObjectPayload JSON boundary wrapper.
- pkg/pluginapi/plugin.go: RegisterSchema JSON boundary.
- pkg/pluginapi/extensions.go: JSON boundary for ExtensionSet payloads.
- pkg/datasetapi/types.go: JSON boundary for RunRequest/Row/RunResult/TemplateRuntime.
- pkg/datasetapi/extensions.go: JSON boundary for ExtensionSet payloads.
- pkg/datasetapi/payload.go: ExtensionPayload JSON boundary wrapper.
- pkg/datasetapi/facade.go: JSON boundary for facade serialization.
- pkg/datasetapi/host_template.go: HostTemplate JSON boundary + internal parameter helpers.
- *_test.go: allow any per policy; guard excludes tests.

## Next steps
- [x] Adopt strict policy: remove public any except JSON/codec boundaries (map[string]any) or wrapper types; confirm alignment with ADR-0003.
- [x] Convert the strict policy into docs/annex/0004-typing-guidelines.md with explicit exceptions and whitelist rules.
- [x] Define guard whitelist scope based on the policy (including how to exclude tests and internal-only helpers).

## API changes (public surface)
- [x] For each disallowed any, choose a concrete type or narrow interface and document trade-offs.
- [x] Replace ExtensionSet Get/Core return types with ObjectPayload/ExtensionPayload; update Raw() to map[string]map[string]map[string]any.
- [x] Replace Change Before/After any with ChangePayload wrapper over json.RawMessage; update internal/core adapter mapping to JSON snapshots without new imports.
- [x] Replace datasetapi Parameter Example/Default any with json.RawMessage plus decode helpers; keep validation behavior intact. **Target removal by 2026-03-31.**
- [x] Update call sites and tests; keep JSON/codec boundaries (map[string]any) for extension payloads per ADR-0003.
- [x] If exported API changes, update snapshots:
  - go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update
  - go test ./pkg/datasetapi -run TestGenerateDatasetAPISnapshot -update

## Guard + linting
- [x] Add a CI guard to fail disallowed any usage (stdlib AST scan or go test guard; no new deps).
- [x] Add internal/ci/any_allowlist.json with file/symbol entries + exclude_globs for tests.
- [x] Wire the guard into make lint and CI required checks.
- [x] Ensure the whitelist mechanism is explicit and reviewed.

## Tests + benchmarks
- [x] Run make lint and make test; add targeted package tests if needed.
- [x] Add microbenchmarks for clone/serialization hot paths in pkg/pluginapi and pkg/datasetapi.
- [x] Replace ad-hoc benchmarking with sweet + benchstat artifacts:
  - [x] Add a pinned Sweet suite (count >= 10) that runs the clone benchmarks.
  - [x] Aggregate Sweet results into a single deterministic `*.results` artifact.
  - [x] Add a `*.withmeta.results` artifact with file-level metadata (commit, Go version, sweet version, runner).
  - [x] Generate a benchstat comparison report (baseline vs PR).
  - [x] Store a single baseline artifact and define the update workflow.
  - [x] Wire CI to publish `*.withmeta.results`, `benchstat.txt`, and Sweet logs as artifacts.
- [x] Guard Sweet patch application with a clear failure message when upstream changes (see `scripts/benchmarks/prepare_sweet.sh`).
- [x] Add an automatic re-diff script for refreshing `scripts/benchmarks/sweet_colonycore.patch` (see `scripts/benchmarks/refresh_sweet_patch.sh`).

## Review + docs
- [ ] Prepare a review checklist for code owners to confirm no reverse dependency on any contracts.
- [ ] Add a changelog entry with typing hardening and migration notes (confirm destination or create file if required).
