# Contributing to ColonyCore

Thanks for considering a contribution — this project keeps things lightweight and welcoming.

## TL;DR
- Fork → small PRs → clear “What/Why/How” → basic tests → maintainer review.
- Follow the repo’s RFCs/ADRs and do not contradict them.
- Agents and automations must follow the guardrails in `AGENTS.md`.
- Be kind and constructive.

## Code of Conduct
Respectful, inclusive communication is expected. See the `CODE_OF_CONDUCT.md` for details.

## How to Contribute
- **Good first issues:** look for the `good first issue` or `help wanted` labels.
- **Bugs:** include repro steps and expected vs actual behavior.
- **Features:** open an issue first to agree on scope, then reference any relevant RFC.
- **Docs:** typo fixes and small clarifications are welcome without prior discussion.

## Workflow
1. **Discuss**: open an issue for non-trivial changes. Link existing RFC/ADR if applicable.
2. **Branch**: create a feature branch in your fork.
3. **Code**: keep PRs focused and small and update docs if behavior changes.
4. **Test**: add or update tests where it makes sense. Run Pre-commit hooks locally before pushing.
5. **PR**: use the short PR template.

## Scoping work
- Identify which layers you will touch (`internal/core`, adapters, persistence drivers, public APIs) and whether the change crosses package boundaries guarded by `.import-restrictions`.
- Note any generated artifacts involved (entity-model sources, SQL/OpenAPI/ERD outputs, schema bundles) and plan to run the matching `make entity-model-verify`/generation targets if affected.
- Look for API snapshots that may need updates (`internal/ci/pluginapi.snapshot`, `internal/ci/datasetapi.snapshot`) and contextual accessor guards in `pkg/pluginapi`/`pkg/datasetapi`.
- For persistence edits, expect parity across memory/sqlite/postgres drivers and their tests under `internal/infra/persistence/**` plus the core adapters that wrap them.
- Default commands to budget for: `make lint`, `make test`, and any generator or adapter-specific suites noted in the README/ADRs.

## Commit and PR Style
- **Commits**: conventional commits are *encouraged*, e.g. `feat: …`, `fix: …`. (see [conventionalcommits.com](https://www.conventionalcommits.org/en/v1.0.0/)).
- **PR title**: short and descriptive, mirroring the main change.
- **PR body**: include “What/Why/How”, test notes, and any breaking changes. Favor concise phrases and titles. The template should guide you through.

## Issues
Use the issue forms:
- **Bug**: repro, logs, expected vs actual.
- **Feature**: problem, proposed solution, alternatives.
- **RFC change (optional)**: link `RFC-xxx` and describe the proposed adjustment.

## Tests
- Add unit/integration tests when adding logic or fixing a bug.
- Manual “operator-like” checks are fine for PoC work; describe steps in the PR.

## Test Utilities
- Shared guard helpers that multiple packages use live in `colonycore/testutil`; keep that package free of plugin or adapter imports so existing import rules stay valid.
- When you need fixtures that reach across architectural boundaries (for example, adapter tests installing reference plugins), create a package-scoped `testutil` directory next to the code under test, document the dependency in `.import-restrictions`, and gate it from production code via guard tests.
- Dataset adapter tests reuse `internal/adapters/testutil` to install the reference frog plugin via `core.Service`; that helper is allowed to touch the persistence memory/sqlite stores and the public dataset/plugin/domain APIs. Keep any new dependencies scoped to those surfaces and extend the `.import-restrictions` file before relying on them.
- Prefer reusing existing helpers before adding new ones, and explain any new cross-package utilities in the PR description so reviewers can confirm boundary expectations.

## RFCs / ADRs
- Architecture and rules live under `docs/rfc/` and `docs/adr/`, as well as `docs/annex/`.
- Changes that affect core contracts or lifecycles should start as an RFC. Do NOT merge features that conflict with accepted RFCs.

## Local Dev
- Use the `README` or `Makefile` for build/test commands. Typical flow:
  - Build: `make build` or language-native build
  - Test: `make test`
  - Lint/format: `make lint` (runs gofmt/vet/registry/golangci plus Ruff and the R lintr)
- Import guardrails rely on `import-boss`; the runbook in `docs/annex/0003-import-boss-runbook.md` covers command syntax and troubleshooting.

## Client Linting
- `make lint` (or `pre-commit run --all-files`) exercises the Go, Python, and R linters exactly as CI does; run it before pushing if you touch `clients/python` or `clients/R`.
- Quick setup:
  1. Install Ruff with `python -m pip install --require-virtualenv -r clients/python/requirements-lint.txt` to match the pinned `0.5.7` version.
  2. Install R ≥ 4.0 (`sudo apt install r-base` on Debian/Ubuntu). `scripts/run_lintr.py` pins `lintr==3.1.2` and `xml2==1.3.6` into `.cache/R-lintr` automatically unless `LINTR_SKIP_AUTO_INSTALL=1` is set.
- Troubleshooting:
  - Ruff missing/mismatched → reinstall using the requirements file; if you manage multiple environments, keep one aligned with 0.5.7 for this repo.
  - R package install errors → install `libcurl4-openssl-dev libxml2-dev libxslt1-dev` and retry, or pre-install the pinned packages with `Rscript -e "remotes::install_version('lintr', version = '3.1.2')"`.
- Auto-fix shortcuts:
  - Python: `python -m ruff check --fix clients/python`; rerun `make python-lint` afterwards.
  - R: `Rscript -e "styler::style_dir('clients/R')"` handles formatting; rerun `make r-lint` afterwards.
- Full details live in `clients/python/LINTING.md` and `clients/R/LINTING.md`; keep those docs in sync when changing lint rules or pinned tool versions.
- Keep platform-specific instructions minimal and script repeatable steps.

## Pre-commit Hooks
- **Prereqs**: install Python 3.11+, GNU make, Go, pnpm (via `corepack enable pnpm`), and R (the hook bootstraps the `lintr` and `xml2` packages automatically when needed).
- **Install hooks**: `pipx install pre-commit` (or `python -m pip install --user pre-commit`) once, then run `pre-commit install --install-hooks`. Re-run the install command after pulling updates to `.pre-commit-config.yaml`.
- **What runs**: `pre-commit run --all-files` matches CI and ensures gofmt/go vet/golangci-lint, Ruff for Python, Prettier 3.3.3 (via `pnpm dlx`) for JS/TS/YAML/Markdown, a local `go mod tidy` guard, R `lintr`, gitleaks secret scanning, the RFC registry check, and OpenAPI validation for `docs/schema/dataset-service.openapi.yaml` using `openapi-spec-validator`.
- **Troubleshooting**: wipe environments with `pre-commit clean`, ensure `golangci-lint`/`Rscript` stay on `PATH`, and let the R hook auto-install `lintr`/`xml2` into `.cache/R-lintr` (`LINTR_SKIP_AUTO_INSTALL=1` if you prefer manual installs). If the install step fails, install the system dependencies (`libcurl4-openssl-dev`, `libxml2-dev`, `libxslt1-dev` on Debian/Ubuntu) or pre-install the R packages yourself. Allow `pnpm` to fetch Prettier the first time it runs, and for OpenAPI lint failures inspect the YAML under `docs/schema/`. Override `PRE_COMMIT_HOME` if you need to share caches across clones.
- **CI**: GitHub Actions runs `make lint` before `pre-commit run --all-files` (skipping the duplicate lint hook) after provisioning Node/pnpm and R, so you get the same checks server-side.
- **Emergency bypass**: prefer `SKIP=<hook id> pre-commit run --all-files` (for example `SKIP=check-jsonschema-openapi`); use `git commit --no-verify` only when absolutely necessary and follow up with a fix before merging.
## Style and Tooling
- Follow existing code style and run formatters/linters where available.
- Write clear, concise comments and docstrings.
- Keep dependencies minimal and explain new ones in the PR.

## Licensing
By contributing, you agree your changes are provided under this repository’s license.

## Security
Do not open public issues for sensitive vulnerabilities. If `SECURITY.md` exists, follow it. Otherwise contact a maintainer privately.

## Maintainers’ Notes (lightweight)
- Protect `main`: require at least one approval to merge.
- Prefer squash-merge: allow maintainer edits on PRs.
- Keep friction low: optimize for contributor happiness and clarity.

---

Happy hacking! If something here blocks you, open an issue and suggest an improvement to this guide.
