# Annex 0007: Catalog Lifecycle Management

- Status: Draft
- Owners: Core Maintainers
- Last Updated: 2026-03-10
- Related: RFC-0001, ADR-0002, ADR-0009, issue #27 (Phase 4)

## Purpose

Define the operator workflow for dataset catalog lifecycle actions:

- register templates in a machine-readable catalog
- deprecate templates with explicit windows
- generate migration plans between versions
- validate the whole catalog and its audit trail

## Command Surface

All commands live under `colony catalog`.

```bash
# Add a template descriptor JSON file to the catalog registry.
go run ./cmd/colony catalog add path/to/template.json

# Deprecate a catalog template with reason + window.
go run ./cmd/colony catalog deprecate frog/frog_population_snapshot@0.1.0 \
  --reason "superseded by 0.2.0" \
  --window-days 90

# Generate a migration plan from old template to new template.
go run ./cmd/colony catalog migrate \
  frog/frog_population_snapshot@0.1.0 \
  frog/frog_population_snapshot@0.2.0

# Validate all catalog entries and the audit hash chain.
go run ./cmd/colony catalog validate
```

Common flags:

- `--catalog`: registry JSON path (default `catalog/catalog.json`)
- `--audit-log`: audit JSONL path (default `catalog/audit.log.jsonl`)
- `--metadata-dir`: metadata output directory (default `catalog/metadata`)
- `--actor`: actor identity for audit entries (defaults to `COLONY_ACTOR`/shell user)

## Data Outputs

### Catalog registry

`catalog/catalog.json` stores registered templates, deprecation windows, and migration records.

### Deprecation metadata

Deprecation creates machine-readable JSON metadata under:

- `catalog/metadata/deprecations/*.json`

### Migration metadata

Migration plan generation creates machine-readable JSON metadata under:

- `catalog/metadata/migrations/*.json`

## Audit Logging Model

Every catalog operation (`add`, `deprecate`, `migrate`, `validate`) appends one JSON line audit entry.

Audit entries include:

- operation name
- actor
- catalog path
- status (`success` or `error`)
- optional structured details
- `prev_hash` + `hash` for hash-chain integrity

The hash chain is validated by `colony catalog validate`; mismatches fail validation.

## Operational Best Practices

- Prefer immutable template versions; add new versions instead of editing existing JSON descriptors.
- Always include a concrete deprecation reason and a bounded window (`--window-days`).
- Treat migration plans as release artifacts and review breaking changes before rollout.
- Run `colony catalog validate` in CI before promotion.
- Persist audit logs in append-only storage and include them in backup/retention policy.

## Failure Modes

`colony catalog validate` returns non-zero when:

- any template descriptor violates `pkg/datasetapi` validation rules
- duplicate template slugs exist in catalog
- deprecation metadata is structurally invalid
- audit hash-chain verification fails

## Notes

- This phase introduces lifecycle controls for dataset catalogs; observability instrumentation is expanded in Phase 5.
- The command intentionally reuses `pkg/datasetapi` template invariants to keep CLI and runtime behavior aligned.
