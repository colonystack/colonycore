# ADR 0001: Migration, Backfill & Rollback Strategy

- Status: Draft
- Deciders: Tobias Harnickell
- Date: 2025-09-21
- Related RFCs: 0001-colonycore-base-module

## Context
ColonyCore requires a repeatable approach for evolving schemas and data while maintaining compliance guarantees and uptime for animal colony operations. Changes often introduce new regulatory fields or audit requirements that must coexist with legacy data during phased rollouts.

## Decision
- Wrap all schema-affecting writes behind feature flags (e.g., `FF_core_schema_v1`) to gate new paths while keeping reads backward compatible.
- Execute forward-only SQL migrations paired with reversible compensating scripts stored under `migrations/rollback/<id>.sql`.
- Run idempotent backfill jobs whose progress is tracked in a `migration_jobs` ledger to support pause/resume and auditing.
- Introduce a feature-flagged dual-write mode when new tables are added, and monitor the behavior before cutting over read paths.
- Define rollback triggers that cover failed conformance tests, SLA breaches, or compliance alerts. When they fire, disable the feature flags, apply compensating migrations, and restore from the latest verified snapshot.
- Maintain migration runbooks with RACI assignments and require Compliance sign-off whenever data classes A/B are affected.

## Consequences
- Provides reversible, auditable migrations aligned with compliance expectations.
- Increases upfront implementation effort due to dual writes and compensating scripts but reduces incident blast radius.
- Requires disciplined feature flag management and observability to detect rollout issues early.
