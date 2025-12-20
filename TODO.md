# TODO: Typing Hardening (any usage)

## References (read/align)
- ARCHITECTURE.md
- docs/adr/0003-core-domain-schema.md
- docs/adr/0009-plugin-interface-stability-and-versioning.md
- docs/adr/0010-contextual-accessor-pattern.md
- CONTRIBUTING.md
- docs/annex/0003-import-boss-runbook.md

## Scope + boundaries
- [ ] Confirm layers/packages in scope (expected: pkg/pluginapi, pkg/datasetapi, docs/).
- [ ] Confirm no cross-layer import changes; update .import-restrictions only if new deps are required.
- [ ] Identify generated artifacts that may need refresh (internal/ci/pluginapi.snapshot, internal/ci/datasetapi.snapshot, entity-model outputs if touched).

## Policy
- [ ] Create docs/architecture/typing-guidelines.md with allowed any usage and an exception/whitelist mechanism.
- [ ] Record trade-offs (tight interfaces vs extensibility) and note compatibility posture per ADR-0009.

## Inventory + classification
- [ ] Enumerate all public any usage in pkg/pluginapi and pkg/datasetapi.
- [ ] Classify each use as allowed boundary vs disallowed; capture a whitelist of allowed locations for guard tooling.

## API changes (public surface)
- [ ] For each disallowed any, choose a concrete type or narrow interface and document trade-offs.
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
