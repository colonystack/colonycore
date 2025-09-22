# Contributing to ColonyCore

Thanks for considering a contribution — this project keeps things lightweight and welcoming.

## TL;DR
- Fork → small PRs → clear “What/Why/How” → basic tests → maintainer review.
- Follow the repo’s RFCs/ADRs and do not contradict them.
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
3. **Code**: keep PRs focused and small; update docs if behavior changes.
4. **Test**: add or update tests where it makes sense.
5. **PR**: use the short PR template.

## Commit and PR Style
- **Commits**: conventional commits are *encouraged*, e.g. `feat: …`, `fix: …`. (see [conventionalcommits.com](https://www.conventionalcommits.org/en/v1.0.0/)).
- **PR title**: short and descriptive, mirroring the main change.
- **PR body**: include “What/Why/How”, test notes, and any breaking changes. The template should guide you through.

## Issues
Use the issue forms:
- **Bug**: repro, logs, expected vs actual.
- **Feature**: problem, proposed solution, alternatives.
- **RFC change (optional)**: link `RFC-xxx` and describe the proposed adjustment.

## Tests
- Add unit/integration tests when adding logic or fixing a bug.
- Manual “operator-like” checks are fine for PoC work; describe steps in the PR.

## RFCs / ADRs
- Architecture and rules live under `docs/rfc/` and `docs/adr/`, as well as `docs/annex/`.
- Changes that affect core contracts or lifecycles should start as an RFC. Do NOT merge features that conflict with accepted RFCs.

## Local Dev
- Use the `README` or `Makefile` for build/test commands. Typical flow:
  - Build: `make build` or language-native build
  - Test: `make test`
  - Lint/format: `make lint`
- Keep platform-specific instructions minimal and script repeatable steps.

## Style and Tooling
- Follow existing code style and run formatters/linters where available.
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
