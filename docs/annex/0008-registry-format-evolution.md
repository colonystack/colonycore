# Annex: Registry Format Evolution & Compatibility Baseline

- Status: Draft
- Last Updated: 2026-03-15
- Owners: Core Maintainers

## 1. Purpose

This annex defines how the governance registry evolves before the first stable
ColonyCore release.

There is no historical released registry line to preserve yet. Instead, the
project establishes a compatibility baseline from the current accepted contract:

- Authoring format: `docs/rfc/registry.yaml`
- Schema contract: `docs/schema/registry.schema.json`
- Validation and fixer entrypoint: `cmd/registry-check`
- Frozen compatibility corpus: `testutil/fixtures/registry/compat`

Future schema work must extend that baseline rather than depending on replaying
older repository commits.

## 2. Current Registry Contract

The registry remains a repo-relative YAML document validated against the JSON
Schema in `docs/schema/registry.schema.json`.

Rules that must remain stable unless a governing RFC or ADR says otherwise:

- top-level `documents:` list with one entry per RFC, ADR, or annex
- canonical document types: `RFC`, `ADR`, `Annex`
- canonical statuses: `Draft`, `Planned`, `Accepted`, `Superseded`, `Archived`
- repo-relative document paths with no leading slash or whitespace
- status parity between `docs/rfc/registry.yaml` and the document front matter or
  status header in the referenced file

## 3. Canonicalization and Fixer Scope

`registry-check` supports limited in-place repair for canonicalizable authoring
issues:

```bash
go run ./cmd/registry-check --fix --registry docs/rfc/registry.yaml
```

The fixer currently normalizes:

- document and linked reference ID casing (`rfc-0001` -> `RFC-0001`)
- document type casing (`adr` -> `ADR`, `annex` -> `Annex`)
- registry status tokens (`draft (working)` -> `Draft`)
- quorum formatting (`1 / 2` -> `1/2`, `Majority` -> `majority`)
- repo-relative path cleanup (`./docs\\rfc\\foo.md` -> `docs/rfc/foo.md`)

The fixer is intentionally narrow:

- it only works on the repository's simple registry YAML shape
- it rewrites canonical field order and drops comments
- it does not invent missing metadata or repair semantically invalid documents
- it must stay safe to run before validation in CI and local workflows

## 4. Compatibility Baseline

`testutil/fixtures/registry/compat` is the forward-compatibility corpus for
registry evolution.

Those fixtures represent canonical, currently supported registry examples. The
suite around them enforces two guarantees:

1. Every compatibility fixture still validates unchanged.
2. The fixer is idempotent on every compatibility fixture.

That baseline is intentionally rooted in the current contract rather than old
commits because ColonyCore has not shipped an initial stable release.

## 5. Evolution Rules

Before `v1.0.0`, prefer additive schema changes and canonicalization fixes over
breaking field or status changes.

Any proposal that changes the registry contract should:

1. update `docs/schema/registry.schema.json`
2. update `docs/rfc/registry.yaml` and any affected governance documents
3. add or refresh fixtures in `valid/`, `invalid/`, `edge/`, and `compat/`
4. document the operator migration path in this annex or a successor ADR/RFC
5. keep `make registry-lint`, `make lint`, `make test`, and `pre-commit run --all-files` green

If a future release introduces an actual backwards-compatibility promise, this
annex becomes the seed for the released migration guide and compatibility suite.

## 6. Operator Workflow

Use the existing validation flow before merge:

```bash
make registry-lint
go test ./cmd/registry-check -run 'TestRegistryFixtures|TestRegistryCompatibilityFixtures' -count=1
```

When the registry fails only on canonicalization issues, run the fixer and
review the resulting diff:

```bash
go run ./cmd/registry-check --fix --registry docs/rfc/registry.yaml
```

If the fixer cannot resolve the issue, update the registry or the linked
document directly and rerun validation.
