# Annex 0006: Dataset Template Validation Specification

- Status: Draft
- Owners: Core Maintainers
- Last Updated: 2026-03-09
- Related: ADR-0009, RFC-0001

## Purpose

Define enforceable invariants for dataset templates registered through
`pluginapi.Registry.RegisterDatasetTemplate` and validated by `colony lint dataset`.

## Required fields

Dataset templates must provide:

- `key` (non-empty)
- `version` (non-empty)
- `title` (non-empty)
- `dialect` (`sql` or `dsl`)
- `query` (non-empty)
- at least one `columns` entry
- at least one `output_formats` entry

Runtime templates additionally require:

- `binder` (non-nil) so the host can construct a runner.

## Validation rules

Validation is performed by `pkg/datasetapi` and fails registration/linting when any
rule is violated.

### Query rendering semantics

- SQL templates must be read-only:
  - query starts with `SELECT` or `WITH`
  - mutating statements (`INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`, `CREATE`,
    `TRUNCATE`, `REPLACE`, `MERGE`) are rejected
- DSL templates must include:
  - a `REPORT` declaration
  - a `SELECT` clause

### Parameters

- Parameter names are required and must match `[A-Za-z][A-Za-z0-9_]*`
- Parameter names must be unique (case-insensitive)
- Supported types: `string`, `integer`, `number`, `boolean`, `timestamp`
- `enum` is allowed only for `string` parameters and values must be non-empty and unique
- `example` and `default` must be valid JSON when provided
- `default` values are type-checked at template validation time

### Columns

- Column names and types are required
- Column names must match `[A-Za-z][A-Za-z0-9_]*`
- Column names must be unique (case-insensitive)

### Output formats

- Allowed: `json`, `csv`, `parquet`, `png`, `html`
- Entries must be unique

### Metadata

- `metadata.entity_model_major`, when present, must be greater than zero

### Descriptor slug

- When `slug` is present, it must equal `plugin/key@version`

## Fail-fast behavior

- Plugin registration fails immediately when a template violates any rule.
- The error is returned as a field-scoped list (`TemplateValidationError`), so plugin
  authors can fix issues before runtime execution.

## CLI tooling

`colony lint dataset` validates JSON template descriptor files using the same
specification.

Usage:

```bash
go run ./cmd/colony lint dataset testutil/fixtures/dataset-templates/valid
go run ./cmd/colony lint dataset --file path/to/template.json
```

The command accepts file and directory paths, recursively scans `.json` files, and
returns non-zero when any template fails validation.
