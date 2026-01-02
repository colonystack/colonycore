# Annex 0005: Documentation Standards

- Status: Draft
- Owners: Core Maintainers
- Last Updated: 2026-01-02
- Related: #114, #115, #116

## Scope

These standards apply to Markdown documentation in this repository, including `docs/`,
top-level guides, and reference material that ships with the codebase.

## Tooling

- Formatting: Prettier (via `pre-commit`).
- Linting: markdownlint (via `make lint-docs`).

Markdownlint is intentionally separated from the default `make lint` flow to avoid
blocking contributors on legacy findings while the baseline is being retired.

## Baseline Workflow

- Baseline file: `internal/ci/markdownlint.baseline.json`.
- Update the baseline with `make lint-docs-update` after intentional rule changes or
  when the existing docs have been cleaned up.
- New findings (issues not listed in the baseline) should be fixed directly; only
  update the baseline for net-new legacy scope changes.

`make lint-docs` reports the count of new findings (for example, "Found 12 new
markdownlint issue(s)") and exits non-zero; any CI job that runs `make lint-docs`
will fail when issues exist outside the baseline.

## Commands

```bash
make lint-docs
make lint-docs-update
```

## Pre-commit Hook

The markdownlint hook is registered as manual-only to preserve default workflows:

```bash
pre-commit run markdownlint-docs --hook-stage manual
```

## Notes

- Consolidation into the main lint flow is tracked in #116.
- GolangCI tooling adjustments for the broader lint stack are tracked in #114.
