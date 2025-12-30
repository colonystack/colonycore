# Benchmarks (Sweet)

This folder wires ColonyCore's Sweet benchmark harness into the upstream
`golang.org/x/benchmarks` repo via a generated patch.

## Patch + overlay workflow

- `scripts/benchmarks/sweet_colonycore.patch` is generated; do not edit it by hand.
- `scripts/benchmarks/sweet_overlays/` is the source of truth for files that must
  stay byte-for-byte in sync with the patch. Use a `.tmpl` suffix for Go files
  so they do not become part of this module (for example,
  `sweet/harnesses/colonycore.go.tmpl`).

## Refreshing the patch

1. Edit files under `scripts/benchmarks/sweet_overlays/` (including any `.tmpl` files).
2. Run `scripts/benchmarks/refresh_sweet_patch.sh` to regenerate
   `scripts/benchmarks/sweet_colonycore.patch`.

## Drift detection

`scripts/benchmarks/prepare_sweet.sh` validates that overlay files exactly match
the patched Sweet tree. If they differ, the script exits with an error so CI
prompts you to refresh the patch.
