# Annex 0003: import-boss Runbook

- Status: Draft
- Linked RFCs / ADRs: Architecture guardrails in `ARCHITECTURE.md`
- Owners: Platform Engineering
- Last Updated: 2025-10-25

## Purpose
`import-boss` enforces the import boundaries that keep ColonyCore’s layered architecture intact. This annex documents the tool’s command-line usage, the structure of `.import-restrictions` files, and the recommended ways to test rule changes efficiently.

## Command-line usage

```bash
# verify all packages that declare .import-restrictions
make import-boss

# targeted run (use Go import paths, not file-system paths)
$(go env GOPATH)/bin/import-boss \
  --verify-only \
  --input-dirs colonycore/internal/core,colonycore/pkg/extension
```

Key flags:

| Flag | Meaning |
| --- | --- |
| `--input-dirs` | Comma-separated list of Go import paths that should be verified. Repeating the flag is equivalent; each argument must be an import path (for example `colonycore/pkg/domain`), **not** a relative directory such as `./pkg/domain`. |
| `--verify-only` | Skip file generation and fail if the current imports violate the rules. |
| `--alsologtostderr`, `--logtostderr`, `--v` | Standard klog controls. Helpful when you need verbose traces while developing a rule. |

CI and `make import-boss` discover every `.import-restrictions` file via `find`, convert each directory into its import path with `go list -m`, and pass the combined list as a single comma-separated value to `--input-dirs`. This ensures new packages are only checked once they opt in by adding a `.import-restrictions` file.

## `.import-restrictions` structure

Restriction files are YAML documents with two top-level arrays:

```yaml
Rules:
  - SelectorRegexp: "^colonycore/"
    AllowedPrefixes:
      - "colonycore/pkg/pluginapi"
    ForbiddenPrefixes:
      - "colonycore/internal/"
InverseRules:
  - SelectorRegexp: "^colonycore/pkg/pluginapi"
    AllowedPrefixes:
      - "colonycore/internal/core"
    ForbiddenPrefixes:
      - "colonycore/plugins"
    Transitive: true
```

- **Rules** constrain what the current package may import. When an import path matches `SelectorRegexp`, the import passes if it matches at least one entry in `AllowedPrefixes` and no entry in `ForbiddenPrefixes`. If an import never matches any selector, it is implicitly allowed.
- **InverseRules** constrain who may import the current package. The same allow/forbid semantics apply, but they are evaluated against *incoming* imports. Set `Transitive: true` to enforce the rule on both direct and transitive dependents; otherwise only direct imports are checked.

Common patterns:

| Use case | Example |
| --- | --- |
| Self-contained packages | Allow only the package’s own prefix in `Rules` so it cannot import anything outside the approved surface. |
| Layer boundaries | List permitted upstream layers in `Rules`; list the downstream layers that may depend on the package in `InverseRules`. |
| Temporary exceptions | Add an additional allowed prefix with a TODO comment; remember to remove it once the dependency is resolved. |

## Behaviour walkthrough

1. For every package in `--input-dirs`, `import-boss` loads every `.import-restrictions` file from the package’s directory up to the module root (`go.mod`). This allows coarse rules at higher directories and fine-grained overrides deeper in the tree.
2. Each import is evaluated against the aggregated `Rules` in order. The first rule whose `SelectorRegexp` matches controls the decision. A match must satisfy at least one allowed prefix and avoid all forbidden prefixes.
3. The tool repeats the process for incoming imports using `InverseRules`. The selector must match the importing package’s path, and the allowed/forbidden prefixes are applied the same way.
4. Any violation results in a multi-line error that lists the failing imports. When multiple packages fail, they are grouped in the output.

If you see apparently “missing” violations, double-check that the package you expect to be guarded has a `.import-restrictions` file and that the path handed to `--input-dirs` is the canonical Go import path.

## Testing rule changes

1. **Unit run** (fast feedback while editing a single file):
   ```bash
   $(go env GOPATH)/bin/import-boss --verify-only --input-dirs colonycore/internal/core
   ```
   Add `--v=4 --logtostderr` to inspect how selectors evaluate.

2. **Repository run** (matches CI behaviour):
   ```bash
   make import-boss
   ```
   The Makefile sets `GOCACHE` to the local `.cache/go-build` directory to avoid permission issues in sandboxed environments.

3. **Negative test** (optional sanity check): temporarily add a blank import that should be forbidden (for example `import _ "colonycore/internal/core"` in a plugin package) and rerun `make import-boss`. The run should fail, confirming the guard still triggers. Remove the test import afterwards.

## Verification checklist

- [ ] Added/updated `.import-restrictions` in every package touched by new dependencies.
- [ ] Ran `$(go env GOPATH)/bin/import-boss --verify-only --input-dirs <package path>` to verify the rule locally.
- [ ] Ran `make import-boss` (or the full `make lint`) and confirmed it passes.
- [ ] For new packages, ensured they provide a `.import-restrictions` file so the Makefile includes them automatically.
- [ ] (Optional) Captured useful `--v` trace snippets in the PR description if the rule change is non-obvious.

Following this workflow keeps the import rules accurate while giving contributors repeatable steps to validate their changes.
