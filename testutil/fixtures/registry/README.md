# Registry Fixtures

These fixtures exercise `cmd/registry-check` against the JSON Schema in `docs/schema/registry.schema.json`.

Structure:

- `valid/`: expected to pass schema + status validation.
- `invalid/`: expected to fail validation.
- `edge/`: valid but unusual inputs (empty lists, status headers).
- `docs/`: stub RFC/ADR/Annex files referenced by the registry YAML.

Notes:

- Paths in fixture registries are repo-root relative, avoid `..` to satisfy `validatePath`, and contain no whitespace or leading slashes to satisfy the schema pattern.
- The registry parser is intentionally minimal: 2-space indentation, list items via `-` (empty lists may use `[]`), and no inline YAML objects.
- Status checks accept either `- Status: Draft` lines or a `## Status` header followed by the status line.
