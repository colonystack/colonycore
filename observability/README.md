# ColonyCore Observability

This directory contains the default observability assets for the observability/tooling scope of issue #27:

- Structured event schema and event catalog
- Grafana dashboard defaults
- Prometheus and Alertmanager alert defaults
- Integration patterns for Prometheus, Grafana, Loki, Elasticsearch, Fluent Bit, and Vector

## Structured Event Schema (v1)

All runtime/CLI events use a single JSON envelope (`schema_version=colonycore.observability.v1`).

| Field | Type | Required | Semantics |
| --- | --- | --- | --- |
| `schema_version` | string | yes | Event schema version (`colonycore.observability.v1`). |
| `timestamp` | RFC3339 timestamp | yes | UTC emission time. |
| `source` | string | yes | Emitter source (`cmd.registry-check`, `cmd.colony.catalog`, `internal.adapters.datasets.handler`, etc.). |
| `category` | string | yes | Event domain (`registry.validation`, `plugin.lifecycle`, `rule.execution`, `catalog.operation`). |
| `name` | string | yes | Stable event name within category. |
| `status` | string | yes | One of `start`, `queued`, `running`, `success`, `error`. |
| `duration_ms` | number | no | Operation wall-clock duration in milliseconds. |
| `error` | string | no | Error summary for failed events. |
| `labels` | object<string,string> | no | Low-cardinality identifiers (`template_id`, `plugin_name`, `export_id`, ...). |
| `measures` | object<string,number> | no | Numeric counters/gauges (`rows_total`, `violations_total`, ...). |

### Example

```json
{
  "schema_version": "colonycore.observability.v1",
  "timestamp": "2026-03-11T08:30:01Z",
  "source": "cmd.registry-check",
  "category": "registry.validation",
  "name": "registry.validate",
  "status": "success",
  "duration_ms": 11.2,
  "labels": {
    "registry_path": "docs/rfc/registry.yaml"
  },
  "measures": {
    "documents_total": 12,
    "documents_validated_total": 12
  }
}
```

## Event Catalog

### Registry Validation

- `registry.validate` (`start|success|error`): end-to-end registry validation run
- `registry.document.validate` (`success|error`): document payload/schema checks
- `registry.document.status` (`error`): status mismatch or status parsing errors

### Plugin Lifecycle

- `plugin.load` (`start|success|error`): plugin installation lifecycle in `internal/core.Service`
- `plugin.registration` (`success|error`): plugin registry contribution phase (rules, schemas, dataset templates)

### Rule Execution

- `rule.evaluate` (`success|error`): one event per rule invocation, with timing and violation counts

### Catalog / Template / Export Operations

- `catalog.add` (`success|error`)
- `catalog.deprecate` (`success|error`)
- `catalog.migrate` (`success|error`)
- `catalog.validate` (`success|error`)
- `catalog.template.run` (`success|error`)
- `catalog.export.create` (`success|error`)
- `catalog.export.enqueue` (`queued|error`)
- `catalog.export.process` (`running|success|error`)

## Monitoring Integration Patterns

### Prometheus + Grafana

1. Ship JSON events to an OpenTelemetry Collector, Vector, or Fluent Bit pipeline.
2. Derive Prometheus metrics from event fields (recommended names):
   - `colonycore_event_count_total{category,name,status,source}`
   - `colonycore_event_duration_ms_bucket{category,name,source,...}` (histogram)
3. Import [Grafana dashboard](./grafana/colonycore-observability-dashboard.json).
4. Load [Prometheus alerts](./prometheus/alerts.yaml) and optional [Alertmanager routes](./alertmanager/routes.yaml).

### Log Analytics (Loki / Elasticsearch)

- Keep raw JSON event lines intact.
- Parse `category`, `name`, `status`, `labels.*`, and `measures.*` into indexed fields.
- Build saved views for:
  - plugin load failures over time
  - rule violation spikes by `labels.rule_id`
  - registry validation error trends
  - export failure reasons grouped by `labels.template_id`

### Log Shippers

- **Fluent Bit**: use `parser json` and ship to Loki/Elasticsearch.
- **Vector**: use `remap` transform to map `labels.*` and `measures.*` into tags/metrics.

## Operational Guidance

- Keep `labels` low-cardinality (IDs and stable keys only).
- Keep PII/secrets out of `labels`, `measures`, and `error`.
- Prefer bounded `error` summaries rather than full payload dumps.
- Event emission is synchronous and lightweight; target overhead is negligible relative to operation latency.
