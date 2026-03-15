# ADR-0006: Observability Stack Architecture

- Status: Accepted
- Deciders: Tobias Harnickell
- Date: 2026-03-15
- Linked RFCs: RFC-0001
- Implemented in: PR #137
  (`feat(observability): add structured events and default dashboard/alerts (#137)`)

## Context

PR #137 introduced the first production observability layer for ColonyCore
before this ADR was finalized. The implementation added a shared structured
event model, JSON event recording, and instrumentation across:

- registry validation in `cmd/registry-check`
- plugin lifecycle and rule execution in `internal/core`
- catalog lifecycle commands in `cmd/colony`
- dataset template execution and export processing in `internal/adapters/datasets`

Without an accepted ADR, the repository had working code and assets but no
governance-level record of the chosen event contract, opt-in model, or
extension policy for later observability phases.

## Decision

Adopt a structured, opt-in event emission model as the observability baseline for ColonyCore.

### Recorder Model

The canonical implementation lives in `internal/observability` and centers on
the `Recorder` interface:

```go
type Recorder interface {
    Record(ctx context.Context, event Event)
}
```

Two baseline recorder implementations are authoritative:

- `NoopRecorder`: default recorder that drops events and performs no external I/O
- `JSONRecorder`: serializes one normalized JSON event per line to an `io.Writer`

Components MUST default to `NoopRecorder` or equivalent no-op behavior unless a
concrete recorder is explicitly wired. Current opt-in paths include:

- `core.WithEventRecorder(...)` for `internal/core.Service`
- `datasets.Handler.Events` for dataset HTTP handlers
- CLI opt-in via `--observability-json` for `cmd/registry-check` and `cmd/colony catalog`

This keeps observability enabled by explicit configuration or wiring choice
rather than by mandatory global side effects.

### Canonical Event Schema

When serialized for operators or external tooling, events use schema version
`colonycore.observability.v1`.

| Field | Meaning |
| --- | --- |
| `schema_version` | Event schema version string; current value `colonycore.observability.v1` |
| `timestamp` | UTC RFC3339 emission timestamp |
| `source` | Emitter identity such as `cmd.registry-check` or `cmd.colony.catalog` |
| `category` | Stable domain category |
| `name` | Stable event name within the category |
| `status` | Lifecycle state for the event |
| `duration_ms` | Optional wall-clock duration in milliseconds |
| `error` | Optional bounded error summary |
| `labels` | Optional low-cardinality string dimensions |
| `measures` | Optional numeric counters, gauges, or totals |

Authoritative schema helpers, constants, and normalization behavior are defined
in `internal/observability/events.go`. The canonical operator-facing catalog and
integration assets live in `observability/README.md`.

### Event Categories and Statuses

The accepted v1 category set is:

- `registry.validation`
- `plugin.lifecycle`
- `rule.execution`
- `catalog.operation`

The accepted v1 status set is:

- `start`
- `queued`
- `running`
- `success`
- `error`

Existing event names documented in `observability/README.md` are part of the
accepted baseline shipped by PR #137, including `registry.validate`,
`plugin.load`, `plugin.registration`, `rule.evaluate`, `catalog.add`,
`catalog.deprecate`, `catalog.migrate`, `catalog.validate`,
`catalog.template.run`, `catalog.export.create`, `catalog.export.enqueue`, and
`catalog.export.process`.

### Metrics, Traces, and Assets

Structured events are the canonical cross-component observability signal for the
current phase. Dashboards and alert defaults under `observability/` assume
operators derive metrics from those events through an external collector or log
pipeline.

Canonical assets:

- `observability/README.md`
- `observability/grafana/colonycore-observability-dashboard.json`
- `observability/prometheus/alerts.yaml`
- `observability/alertmanager/routes.yaml`

Process-local helpers such as `internal/core/observability_exporters.go` remain
valid for local metrics and tracing use, but they do not replace the structured
event contract recorded by this ADR.

### Governance and Extension Policy

Future observability work MUST preserve the following governance rules:

- New event categories require an ADR-0006 update because categories define
  long-lived architectural domains.
- New event names within an existing category MUST update
  `observability/README.md` and the relevant tests in the emitting package.
- Breaking schema changes require a new `schema_version` value and an ADR
  update. Additive optional fields may remain within
  `colonycore.observability.v1` if existing field meanings do not change.
- `labels` MUST remain low-cardinality and use stable identifiers or coarse classifications only.
- PII, secrets, tokens, raw payload dumps, and unbounded free-form values MUST
  NOT be emitted in `labels`, `measures`, or `error`.
- Error text SHOULD be bounded summaries suitable for logs and metric derivation
  rather than full-stack traces or record payloads.

## Rationale

1. A single structured envelope provides one interoperable signal across CLI
   tooling, service orchestration, rules, and dataset flows.
2. Opt-in recording avoids imposing external dependencies or noisy output on
   local and test workflows by default.
3. JSON lines are easy to route into Prometheus derivation pipelines, Loki,
   Elasticsearch, OpenTelemetry Collector, Vector, or Fluent Bit without
   coupling the core to a specific backend SDK.
4. The approach shipped in PR #137 already demonstrates useful coverage without
   changing domain contracts or storage choices defined elsewhere in RFC-0001
   and the accepted ADR set.

## Consequences

### Positive

- Default behavior is effectively zero-overhead for event emission because the baseline recorder is `NoopRecorder`.
- Operators can enable structured JSON event streams only where needed,
  including CLI invocations via `--observability-json` and runtime integrations
  via explicit recorder wiring.
- The schema, assets, and implementation are now anchored to canonical
  locations: `internal/observability` and `observability/`.

### Trade-offs and Requirements

- Metrics such as Prometheus counters and histograms are not produced
  automatically from the event stream; operators must run an external
  log-to-metrics or collector pipeline and load the matching rules and
  dashboard assets.
- Because recording is opt-in, deployments that do not wire a recorder will not
  emit structured events and therefore will not feed the provided dashboards or
  alerts.
- Event quality depends on governance discipline: uncontrolled labels, schema
  drift, or payload leakage would reduce operational value and increase risk.

### Follow-up Constraints

- Future observability phases such as additional categories, richer tracing
  correlation, or data-quality events should be gated on this ADR remaining the
  governing baseline.
- Any work that changes the schema contract or category taxonomy must reconcile
  code, tests, and `observability/README.md` before acceptance.

## Alternatives Considered

1. Direct backend-specific instrumentation first, for example hard-coding
   Prometheus or OpenTelemetry SDK usage everywhere. Rejected because it would
   couple the core too early to one stack and complicate CLI reuse.
2. Ad hoc log strings without a typed event envelope. Rejected because metrics
   derivation, dashboards, and governance would remain ambiguous.
3. Always-on JSON event emission. Rejected because the default operator and
   test experience should remain quiet unless observability is explicitly
   requested.

## References

- `docs/rfc/0001-colonycore-base-module.md`
- `internal/observability/events.go`
- `observability/README.md`
- PR #137 (`feat(observability): add structured events and default dashboard/alerts (#137)`)
