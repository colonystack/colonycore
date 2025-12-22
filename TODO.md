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
- [ ] Confirm no cross-layer import changes; update .import-restrictions only if new deps are required.
- [ ] Identify generated artifacts that may need refresh (internal/ci/pluginapi.snapshot, internal/ci/datasetapi.snapshot, entity-model outputs if touched).
- [ ] If Change payloads are narrowed, confirm internal/core adapter updates are in scope and keep import direction intact.

## Policy
- [x] Create docs/annex/0004-typing-guidelines.md with allowed any usage and an exception/whitelist mechanism.
- [x] Record trade-offs (tight interfaces vs extensibility) and note compatibility posture per ADR-0009.

## Inventory + classification
- [x] Enumerate all public any usage in pkg/pluginapi and pkg/datasetapi.
- [ ] Classify each use as allowed boundary vs disallowed; capture a whitelist of allowed locations for guard tooling.
  - [ ] pkg/pluginapi/views.go: Attributes/CoreAttributes/EnvironmentBaselines/Data/ChainOfCustody return map[string]any or []map[string]any (JSON/extension payload boundary).
  - [ ] pkg/pluginapi/extensions.go: ExtensionSet Get/Core return any; Raw is map[string]map[string]any (decision: replace with payload wrappers and map[string]map[string]map[string]any).
  - [ ] pkg/pluginapi/domain_aliases.go: Change/ChangeBuilder before/after and Before()/After() use any (decision: replace with ChangePayload wrapper over map[string]any).
  - [ ] pkg/pluginapi/plugin.go: RegisterSchema schema map[string]any (JSON schema boundary).
  - [ ] pkg/pluginapi/payload.go: ObjectPayload map[string]any (JSON payload boundary).
  - [ ] pkg/datasetapi/types.go: Parameter Example/Default any; RunRequest Parameters map[string]any; Row map[string]any; RunResult Metadata map[string]any; TemplateRuntime ValidateParameters/Run map[string]any (decision: replace Example/Default with ParameterValue or json.RawMessage; others remain JSON boundary).
  - [ ] pkg/datasetapi/extensions.go: ExtensionSet Get/Core return any; Raw is map[string]map[string]any (decision: replace with payload wrappers and map[string]map[string]map[string]any).
  - [ ] pkg/datasetapi/payload.go: ExtensionPayload map[string]any (JSON payload boundary).
  - [ ] pkg/datasetapi/facade.go: Attributes/CoreAttributes/EnvironmentBaselines/PairingAttributes/Data/ChainOfCustody serialization uses map[string]any or []map[string]any (extension payload boundary).
  - [ ] pkg/datasetapi/host_template.go: parameter helpers return any internally (should remain internal-only; confirm guard excludes).
  - [ ] Confirm *_test.go any usage stays test-only or is explicitly excluded from guard scope.

## Next steps
- [ ] Adopt strict policy: remove public any except JSON/codec boundaries (map[string]any) or wrapper types; confirm alignment with ADR-0003.
- [x] Convert the strict policy into docs/annex/0004-typing-guidelines.md with explicit exceptions and whitelist rules.
- [ ] Define guard whitelist scope based on the policy (including how to exclude tests and internal-only helpers).

## API changes (public surface)
- [ ] For each disallowed any, choose a concrete type or narrow interface and document trade-offs.
- [ ] Replace ExtensionSet Get/Core return types with ObjectPayload/ExtensionPayload; update Raw() to map[string]map[string]map[string]any.
- [ ] Replace Change Before/After any with ChangePayload wrapper; update internal/core adapter mapping from domain.Change to payload map without new imports.
- [ ] Replace datasetapi Parameter Example/Default any with ParameterValue or json.RawMessage plus decode helpers; keep validation behavior intact.
- [ ] Update call sites and tests; keep JSON/codec boundaries (map[string]any) for extension payloads per ADR-0003.
- [ ] If exported API changes, update snapshots:
  - go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update
  - go test ./pkg/datasetapi -run TestGenerateDatasetAPISnapshot -update

## Guard + linting
- [ ] Add a CI guard to fail disallowed any usage (stdlib AST scan or go test guard; no new deps).
- [ ] Wire the guard into make lint and CI required checks.
- [ ] Ensure the whitelist mechanism is explicit and reviewed.

## Tests + benchmarks
- [ ] Run make lint and make test; add targeted package tests if needed.
- [ ] Identify hot paths impacted (clone/serialization/extension handling) and run or add go test -bench baselines; record <=2% regression or waiver.

## Review + docs
- [ ] Prepare a review checklist for code owners to confirm no reverse dependency on any contracts.
- [ ] Add a changelog entry with typing hardening and migration notes (confirm destination or create file if required).
