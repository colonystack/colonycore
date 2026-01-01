# Agent Guidance for ColonyCore

This document is the single entrypoint for any non-human contributor (“agent”) operating in this repository. Treat it as a router to authoritative sources: prefer following the linked documentation over duplicating it here. If any referenced material is missing or inaccessible, pause and request human direction instead of proceeding.

## Definitions and Scope
- **Agent**: any non-human system capable of interpreting and acting on user instructions and the referenced documentation (for example, IDE coding copilot features, CLI assistants, CI/CD automations, or external integrations).
- **Reactive agents** run locally alongside a human operator and act only in response to direct user prompts during an interactive session (for example, an IDE chat or terminal assistant).
- **Proactive agents** initiate work without an in-session human (for example, scheduled or event-driven bots that open PRs or push branches).

All agents are subject to the same repository policies as human contributors and must stay within the explicit operating principles and scopes below.

## Agent Operating Principles
- Read and understand `ARCHITECTURE.md` before you start any change
- Read and understand the RFCs and ADRs before you start any change
- Read and understand `CONTRIBUTING.md` before you start any change
- Read and understand the pre-commit workflow and its references before you start any change
- Obey only documented workflows and user instructions, unless forbidden and not explicitly approved by the user.
- Remain solution-agnostic and state trade-offs.
- Reference authoritative docs rather than duplicating content.
- Explicit human approval may be requested and may override forbidden actions
- In case of conflicting information, the following inputs take precedence:
  - Explicit human approval over `RFC`'s/`ADR`'s, `ARCHITECTURE.md`, `CONTRIBUTING.md`, `README.md` and `AGENTS.md`
  - `RFC`'s/`ADR`'s over `ARCHITECTURE.md`, `CONTRIBUTING.md`, `README.md` and `AGENTS.md`
  - `ARCHITECTURE.md` over `CONTRIBUTING.md`, `README.md` and `AGENTS.md`
  - `CONTRIBUTING.md` over `README.md` and `AGENTS.md`
  - `README.md` over `AGENTS.md`
  - Any document in the status "Draft" is preceded by any document in the status "Approved"

## Authoritative References
Consult these sources before performing work, and link to them in your interaction trail instead of copying excerpts:
- `README.md`: high-level project overview, build workflow, and storage defaults.
- `CONTRIBUTING.md`: contribution guardrails, review expectations, conventional commits, concise writing guidance, and lint/test requirements (including `make lint` and `make test`).
- `ARCHITECTURE.md`: system design, core boundaries, and domain contracts.
- `Makefile`: canonical commands for build, lint, test, registry validation, and pre-commit parity.
- `docs/adr/`, `docs/rfc/`, `docs/annex/`, `docs/schema/`: history of accepted decisions, active proposals, operational runbooks, and canonical schemas. Read relevant items in full, paying attention to “Draft” versus “Approved” status.

If any of these references conflict with a user request, seek clarification before acting unless an explicit human approval resolves the conflict.

## Scoping prompts for agents
- Ask which layers or packages are in play (core services, adapters, persistence drivers, public APIs) and whether the work crosses import-restricted boundaries.
- Surface generated inputs/outputs early: entity-model sources, SQL/OpenAPI/ERD artifacts, and API snapshots in `internal/ci/{pluginapi,datasetapi}.snapshot` often require regeneration.
- Highlight persistence implications from ADR-0007: changes should align memory/sqlite/postgres drivers and keep env-based driver selection intact.
- Flag contextual accessor and architecture guards in `pkg/pluginapi`/`pkg/datasetapi` plus import-boss rules that will fail if boundaries drift.
- Set expectations on commands to run (`make lint`, `make test`, relevant generator or adapter suites) so the user understands the effort and runtime.
- If Docker-backed tasks fail, verify required images are available and consider whether elevated privileges are needed before choosing a path forward.

## Allowed Changes by Agents
- Localized bug fixes in one package or bounded context with tests
- Missing or improved tests that do not change behaviour
- Doc clarifications tied to changed code or doc typo/link fixes
- Non-functional refactors confined to one package/module with tests by path glob and with no public API change
- Formatting changes enforced by repo formatters

Scope boundaries apply cumulatively: stay within the targeted package or module, update affected documentation, and ensure all automated checks pass (`make lint`, `make test`, and relevant pre-commit hooks).

## Forbidden Without Human Approval
- Public API or wire-format changes without RFC/ADR draft and human approval
- Storage or backend switches without RFC/ADR draft and human approval
- Adding or removing dependencies without rationale and human approval
- Edits to CI/`CODEOWNERS.md`/`MAINTAINERS.md`/`SECURITY.md`/RFCs/ADRs without a matching, justified work item and human approval
- Cross-cutting refactors across packages or modules
- Creating, modifying or deleting files or git items outside the PR's targeted package/module or documented change.
- Direct commits to protected branches in upstream
- Git history rewrites of any kind
- Any action if any of the referenced documentation is missing and no bootstrap environment is evident
- Any action, should `AGENTS.md` conflict with any of the referenced docs
- Implementing security-related changes, secrets or permissions without rationale and human approval
- Adding dependencies without rationale and human approval
- Any changes to the license
- Any feature flag default behaviour changes without RFC/ADR draft and human approval
- Schema/DB migrations without rationale and human approval
- Any security posture changes (RBAC, network, auth) without rationale and human approval
- Any change or ambiguity bordering on a forbidden definition without human approval

When in doubt, assume a change is forbidden until a maintainer explicitly approves it.

## Approval Flows
**Reactive agents**  
- Human approval can only be granted through an explicit request in the active chat or session where the agent operates.  
- Capture the approval verbatim in the session transcript or logs and reference it in the PR description or final summary.  
- Stop immediately if the user is silent or the approval is ambiguous.

**Proactive agents**  
- Human approval requires adding the `needs-human-approval` label to the pull request and securing approval from at least one maintainer group entry defined in `CODEOWNERS`.  
- Proactive PRs must include the checklist item “I strictly followed `AGENTS.md`” marked as complete before requesting review.  
- Do not merge or update protected branches until the approval is recorded and the PR is explicitly greenlit by the designated owners.

Document any approval context directly in commit messages or PR discussion threads to preserve traceability.

## Ownership and Compliance
- `AGENTS.md` is owned by the repository owners defined in `CODEOWNERS`; modifications require their review.  
- Agents must leave clear audit trails (commands run, tests executed, links referenced) to support review.  
- If pre-commit hooks, `make lint`, or `make test` fail, stop and request guidance rather than attempting workarounds that violate the rules above.

## Reporting Issues
If you detect missing documentation, conflicting requirements, or gaps in these policies, halt your work and raise it with a human maintainer. Future refinements may add module-specific guidance; until then, defer to the documents linked here and the operating principles above.
