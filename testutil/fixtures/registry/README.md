# Registry Fixtures

These fixtures exercise `cmd/registry-check` against the JSON Schema in `docs/schema/registry.schema.json`.

Structure:

- `valid/`: expected to pass schema + status validation.
- `invalid/`: expected to fail validation.
- `edge/`: valid but unusual inputs (empty lists, status headers).
- `compat/`: frozen canonical registries that define the
  forward-compatibility baseline.
- `docs/`: stub RFC/ADR/Annex files referenced by the registry YAML.

Tooling:

- `go test ./cmd/registry-check -run TestRegistryFixtures -count=1`
  auto-discovers all `*.yaml` fixtures in `valid/`, `edge/`, and `invalid/`.
- `go test ./cmd/registry-check -run TestRegistryCompatibilityFixtures -count=1`
  asserts every `compat/` fixture still validates unchanged and remains
  fixer-idempotent.
- Expectations are directory-driven: `valid/` and `edge/` must pass; `invalid/` must fail.
- Compatibility fixtures are stricter than `valid/`: they are treated as
  frozen canonical examples for future schema evolution work.
- Every invalid fixture must include a sidecar file at
  `<fixture>.yaml.error.txt` containing a non-empty error substring asserted by
  the test.
- Sidecar substrings must be field/value specific (for example
  `$.documents[0].quorum` or `missing required property "title"`), not generic
  phrases like `not in enum` or `does not match pattern`.
- Valid and edge fixtures must not include `.error.txt` sidecars.

Conventions:

- Paths in fixture registries are repo-root relative, avoid `..` to satisfy
  `validatePath`, and contain no whitespace or leading slashes to satisfy the
  schema pattern.
- The registry parser is intentionally minimal: 2-space indentation, list items
  via `-` (empty lists may use `[]`), and no inline YAML objects.
- Status checks accept either `- Status: Draft` lines or a `## Status` header followed by the status line.
