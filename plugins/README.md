# Plugin API (v1)

This guide is the developer-facing contract for ColonyCore plugins. It complements [ADR-0009](../docs/adr/0009-plugin-interface-stability-and-versioning.md) and [RFC-0001](../docs/rfc/0001-colonycore-base-module.md).

## Stable v1 interface surface

The stable plugin entrypoint is `pkg/pluginapi`.

| Interface | Required methods | Purpose |
| --- | --- | --- |
| `pluginapi.Plugin` | `Name() string`, `Version() string`, `Register(pluginapi.Registry) error` | Declares plugin identity and registers plugin contributions with the host. |
| `pluginapi.Registry` | `RegisterSchema(entity string, schema map[string]any)`, `RegisterRule(rule pluginapi.Rule)`, `RegisterDatasetTemplate(template datasetapi.Template) error` | Host callback surface used during plugin registration. |
| `pluginapi.Rule` | `Name() string`, `Evaluate(ctx context.Context, view pluginapi.RuleView, changes []pluginapi.Change) (pluginapi.Result, error)` | Rule hook run by the host during transactional validation. |
| `pluginapi.VersionProvider` | `APIVersion() string` | Reports the currently supported plugin API major (currently `v1` through `pluginapi.GetVersionProvider().APIVersion()`). |

## Plugin lifecycle

1. Host selects a plugin instance and calls `Plugin.Register`.
2. Plugin uses `Registry` to register schemas, rules, and optional dataset templates.
3. Registered rules are invoked by the core rules engine via `Rule.Evaluate`.
4. `RuleView` exposes read-only state; rule outcomes are returned as `pluginapi.Result`.

## Error-handling contract

- `Plugin.Register` must return an error if registration fails (for example, `RegisterDatasetTemplate` returns an error).
- Rules should return:
  - `error` for execution failures (invalid runtime dependencies, unexpected runtime conditions).
  - `pluginapi.Result` violations for policy decisions.
- Avoid panics for validation outcomes. Use violations (`block`, `warn`, `log`) in `pluginapi.Result`.

## Dataset template best practices

Reference spec: `docs/annex/0006-dataset-template-validation.md`.

- Treat dataset templates as read models: keep SQL queries read-only (`SELECT`/`WITH` only).
- Keep `key`, `version`, and `title` stable; changes should be additive and versioned.
- Prefer explicit parameter contracts:
  - use canonical types (`string`, `integer`, `number`, `boolean`, `timestamp`)
  - provide `default` values only when they are valid for the declared type
  - keep parameter names predictable and machine-friendly (`snake_case` style works best)
- Keep column names deterministic and unique to avoid downstream renderer ambiguity.
- Declare only formats the runner can actually materialize.
- Run lint early during development:
  - `go run ./cmd/colony lint dataset testutil/fixtures/dataset-templates/valid`
- Expect fail-fast registration: malformed templates are rejected during `Register`, not first runtime execution.

## Migrating pre-v1 plugins to the accepted v1 contract

There was no released external `v0` plugin compatibility line for ColonyCore.
Treat any pre-stability plugin as a source migration to the accepted `v1`
contract, not as a binary-compatibility target.

Use this migration path:

1. Replace raw constant comparisons with contextual accessors from
   [ADR-0010](../docs/adr/0010-contextual-accessor-pattern.md).
   - Examples: `housing.GetEnvironmentType().IsAquatic()`,
     `organism.GetCurrentStage().IsAdult()`,
     `protocol.GetCurrentStatus().IsApproved()`.
2. Keep registration logic inside `Plugin.Register` limited to the stable
   `pkg/pluginapi.Registry` callbacks.
   - Schema, rule, and dataset template registration should fail fast and return
     errors instead of deferring invalid state to runtime.
3. Update rule hooks to return `pluginapi.Result` violations for policy
   outcomes and `error` only for execution failures.
4. Verify dataset templates against the dataset template spec before release.
   - Run `go run ./cmd/colony lint dataset <dir>` for local fixtures.
5. Run the shared conformance suite before merge:
   - `make plugin-conformance`

This repository treats that migration path as the foundation for any future
documented `v0 -> v1` upgrade notes. If a prerelease plugin must keep both code
paths temporarily, keep the compatibility shim in the plugin repository rather
than widening the accepted `v1` core API.

## Compatibility matrix

Keep this matrix aligned with ADR-0009 when host releases or API majors change.

| ColonyCore host version | Supported plugin API version | Notes |
| --- | --- | --- |
| `v0.1.x` | `v1` | Initial stable plugin API line. |
| `v0.2.x` | `v1` | Contextual accessor expansion kept API major `v1`. |
| `v0.3.x` | `v1` | Current contract line in this repository. |
| `v1.x` (planned) | `v1` | Major host release keeps API `v1` unless ADR-0009 introduces `v2`. |

## Conformance suite

Conformance tests live in `plugins/conformance` and currently run against the reference frog plugin.

Run locally:

```bash
make plugin-conformance
```

What the suite verifies:

- plugin initialization (`Name`, `Version`, successful `Register`)
- schema/rule/template registration expectations
- rule hook behavior against deterministic fixtures
- registration error propagation (dataset registration failures)

### Requirements for external plugin developers

- Match the `pkg/pluginapi` v1 interfaces documented above.
- Add your plugin to the `[]pluginCase` slice in
  `plugins/conformance/plugin_conformance_test.go` (or mirror the same checks
  in your plugin repository).
- Keep CI configured to fail on conformance test failures before merge.
