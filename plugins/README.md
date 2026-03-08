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
- Add your plugin to the conformance suite table in `plugins/conformance/plugin_conformance_test.go` (or mirror the same checks in your plugin repository).
- Keep CI configured to fail on conformance test failures before merge.
