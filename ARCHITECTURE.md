# ColonyCore Architecture

This document summarizes the enforced architectural boundaries of the repository and points to the canonical specifications that govern them. It is intentionally high-level to avoid duplicating content already captured in RFCs, ADRs, and annexes.

## Primary References

- RFC-0001 “ColonyCore Base Module” (`docs/rfc/0001-colonycore-base-module.md`) — system capabilities and lifecycle.
- ADR-0003 “Core Domain Schema” (`docs/adr/0003-core-domain-schema.md`) — authoritative domain entity definitions.
- ADR-0009 “Plugin Interface Stability & Semantic Versioning Policy” (`docs/adr/0009-plugin-interface-stability-and-versioning.md`) — public extension surface and compatibility guarantees.
- ADR-0010 “Contextual Accessor Pattern Implementation” (`docs/adr/0010-contextual-accessor-pattern.md`) — contextual interfaces that decouple plugins from internal constants.
- ADR-0008 “Object Storage Contract” (`docs/adr/0008-object-storage-contract.md`) — blob subsystem expectations referenced by the dataset adapters.

Consult those documents for detailed rationale and background before modifying architectural boundaries.

## Layering & Dependency Direction

ColonyCore follows a hexagonal-inspired layering model:

| Layer | Representative Packages | Notes |
| --- | --- | --- |
| **Domain** | `pkg/domain` | Pure Go entities, rule interfaces, and persistence contracts. No imports from `internal/**` (enforced by `pkg/domain/architecture_test.go` and import-boss). |
| **Application / Core** | `internal/core`, `internal/adapters/datasets` | Orchestrates use cases, rules engine composition, dataset orchestration, plugin installation. Guards forbid type aliases and enforce transaction wiring (`internal/core/alias_guard_test.go`, `internal/core/service_contract_test.go`). |
| **Infrastructure** | `internal/infra/persistence/*`, `internal/infra/blob/*`, `internal/blob` | Backend-specific implementations that satisfy domain interfaces. Import direction is one-way: infrastructure depends on domain, never the reverse. |
| **Extensions** | `pkg/pluginapi`, `pkg/datasetapi`, `plugins/*` | Stable external surface consumed by runtime plugins and dataset binders. |

Dependency direction is validated by:

- `go test ./...` (includes AST-based guards that reject forbidden imports and alias usage).
- `.import-restrictions` in each layer (executed via `make lint` → `make import-boss`).
- `scripts/validate_plugin_patterns.go` (executed from `make lint`) ensuring plugins stay within sanctioned APIs.

Whenever you introduce a new package, add a `.import-restrictions` file and—if needed—a focused guard test mirroring the existing patterns.

## Extension Points

### Plugin Runtime Surface

The plugin API is defined in `pkg/pluginapi` and versioned per ADR-0009. Plugins are expected to consume contextual interfaces and opaque reference types introduced in ADR-0010. Key extension flows:

1. Plugins register via `pluginapi.Plugin`/`Registry`.
2. `internal/core.PluginRegistry` adapts plugin contributions into domain rule implementations and dataset templates.
3. Dataset templates exposed through `pkg/datasetapi` provide contextual access to runtime dependencies and binding hooks.

Future additions to the public surface require an accompanying RFC/ADR update and a versioning decision (see ADR-0009).

As of the v0.3.0 compliance entity expansion, the plugin API exposes dedicated views (`FacilityView`, `TreatmentView`, `ObservationView`, `SampleView`, `PermitView`, `SupplyItemView`) and contextual providers (`FacilityContext`, `TreatmentContext`, `ObservationContext`, `SampleContext`, `PermitContext`, `SupplyContext`). These follow the ADR-0010 pattern (opaque references with semantic helpers) and are catalogued for plugin authors in `docs/plugins/upgrade-notes.md`. Rules MUST rely on the contextual helpers (`GetZone`, `GetCurrentStatus`, `IsActive`, `RequiresReorder`, etc.) rather than string comparisons to remain forward compatible.

#### Architecture Guards for Plugins

The enforcement stack is summarized here:

1. **Import-Boss Restrictions (compile-time)**  
   - Enforced by `.import-restrictions` files under `plugins/`, `pkg/pluginapi`, and `pkg/datasetapi`.  
   - Prevents imports of `internal/**` or `pkg/domain` from plugins.  
   - Run via `make lint` (calls `make import-boss`).

2. **Static Analysis Guards (test-time)**  
   - `pkg/pluginapi/architecture_guard_test.go` checks that contextual interfaces only expose Ref types and retain sentinel marker methods.  
   - `pkg/pluginapi/architecture_ci_test.go` snapshots the exported API surface to detect breaking changes.

3. **Anti-pattern Detection (test-time)**  
   - `pkg/pluginapi/plugin_antipattern_test.go` and `internal/validation` identify legacy usage (raw constants, string comparisons, direct entity equality, etc.).  
   - `scripts/validate_plugin_patterns.go` executes the same heuristics against each plugin during `make lint`.

4. **API Stability Verifiers (CI)**  
   - `pkg/pluginapi/breaking_change_enforcement_test.go` and `pkg/pluginapi/api_snapshot_test.go` guard semantic version expectations described in ADR-0009.

Recommended workflows:

```bash
# Run all lint + guard suites (import rules, plugin pattern validator, golangci-lint)
GOCACHE=$PWD/.cache/go-build make lint

# Run package-level architecture tests only
go test ./pkg/pluginapi -run "Architecture|AntiPattern|Snapshot" -count=1
```

When adding new contextual interfaces or reference types:

1. Extend the provider/contextual interface implementations in `pkg/pluginapi` / `pkg/datasetapi`.
2. Update the guard tests listed above to assert the new semantics.
3. Document the behavior in ADR-0010 (or the relevant ADR/RFC) instead of duplicating lengthy explanations here.

### Dataset Catalog & Exporters

Dataset templates and exporters live in `internal/adapters/datasets`. They adapt plugin-supplied templates (via `DatasetTemplate`) into HTTP handlers and asynchronous export workers. Key points:

- The adapter layer consumes `pkg/datasetapi` only; it never imports plugin code directly.
- Export storage contracts and blob implementations follow ADR-0008. See `internal/blob` and `internal/infra/blob/*` for concrete factories.
- Dataset catalog HTTP handlers rely on the service layer (`internal/core.Service`) rather than persistence directly, preserving the hexagonal boundary.
- Compliance entities are now part of the `TransactionView` surface (`ListFacilities`, `ListTreatments`, `ListObservations`, `ListSamples`, `ListPermits`, `ListSupplyItems`) and are accompanied by `Find*` helpers. Dataset authors should audit templates and ensure contextual access is used when cross-referencing the new entities; see `docs/plugins/upgrade-notes.md` for migration checklists.

Run `go test ./internal/adapters/datasets` after changing dataset adapters to exercise guard branches, HTTP flows, and export worker invariants.

## Enforcement Quick Reference

| Concern | Guard Mechanism | Command |
| --- | --- | --- |
| Forbidden imports / reversed dependencies | `.import-restrictions`, `AssertNoDirectImports`, `AssertNoTransitiveDependency` | `make lint` |
| Type alias regression in core | `internal/core/alias_guard_test.go` | `go test ./internal/core -run TestNoTypeAliases` |
| Plugin architecture compliance | Tests + validator described above | `make lint`, `go test ./pkg/pluginapi ...` |
| API stability snapshots | `pkg/pluginapi/api_snapshot_test.go`, dataset API snapshots | `go test ./pkg/pluginapi`, `go test ./pkg/datasetapi` |

Always run `make lint` and `make test` locally before submitting changes that touch architectural boundaries.
