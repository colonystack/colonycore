# MAINTAINERS

This file lists project maintainers, responsibilities, and lightweight processes.

## Maintainers
- @TobyTheHutt  — role: Maintainer, Admin — contact: tobias@harnickell.ch

> Update this list via PR; at least one existing maintainer must approve maintainer changes.

## Responsibilities
- Triage: label new issues/PRs, close duplicates, request missing info.
- Review: ensure changes follow RFCs/ADRs and repo rules; keep scope tight.
- Merge: approve when CI green and minimal template requirements are met.
- Releases: tag versions, draft notes, create GitHub Releases as needed.
- Security: handle private reports and coordinate fixes.

## Decision Making
- Default: lazy consensus after 48h on non-breaking changes.
- Breaking or RFC-impacting changes: open or reference an RFC; require 2 maintainer approvals.
- Ties: Lead Maintainer decides; document rationale in the PR.

## PR Policy
- Require 1 maintainer approval on `main`; squash-merge preferred.
- Authors may not self-approve for breaking or RFC-impacting changes.
- Keep PRs small and follow the PR template.

## Issue Triage (targets, not guarantees)
- First response: 3–5 days.
- Priorities: `bug` > `feat` > `enhancement` > `docs/chore`.

## Releases
- Tag `vX.Y.Z`
- Patch: fixes only. Minor: backward-compatible features. Major: breaking changes with upgrade notes.

## Becoming a Maintainer
- Show consistent contributions (code, reviews, triage).
- Be nominated by a maintainer. Approval by two maintainers or one admin.

## Code of Conduct
- See `CODE_OF_CONDUCT.md`. Violations may lead to loss of maintainer status.

## Security Contact
- Prefer private email to a maintainer. avoid filing public issues for sensitive reports.

## Bus Factor
- At least two maintainers or one admin must have release and settings access.
