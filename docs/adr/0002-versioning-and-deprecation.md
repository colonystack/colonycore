# ADR 0002: Versioning & Deprecation Policy

- Status: Draft
- Deciders: Tobias Harnickell and SDK Maintainers
- Date: 2025-09-21
- Related RFCs: 0001-colonycore-base-module

## Context
ColonyCore exposes APIs, schemas, and plugin contracts that must evolve without disrupting regulated operations. A consistent versioning and deprecation policy enables predictable upgrades for facilities and species module maintainers.

## Decision
- Adopt semantic versioning (`MAJOR.MINOR.PATCH`) for the core platform. Bump the MAJOR version when introducing breaking API or schema changes and provide at least six months of notice.
- Require species plugins to declare a compatibility matrix (`core >=x.y,<z.0`) and ship automated compatibility tests.
- Follow a deprecation lifecycle: announce via release notes and deprecation guides → supply a dual path (feature flag or backwards-compatible API) → monitor usage telemetry → remove once usage is below 5% or the notice window expires.
- Maintain `deprecations.md` as the single source of truth for deprecation state (announced, in-sunrise, removed) with an assigned owner.
- Emit compile-time warnings and `@deprecated` annotations from SDK code generators for soon-to-be-removed capabilities.

## Consequences
- Establishes clear expectations for external integrators and regulatory auditors regarding change management.
- Requires continuous telemetry to track usage of deprecated features and enforce removal thresholds.
- Encourages early communication and documentation overhead for every breaking change proposal.

## Related ADRs
- ADR-0005: Plugin Packaging & Distribution (packaging implications for versioning & deprecation windows)
- ADR-0007: Storage Baseline (impacts schema migration cadence)
- ADR-0008: Object Storage Contract (adds new API surface governed by this policy)
- ADR-0009: Plugin Interface Stability & Semantic Versioning Policy (specialization of this ADR for plugin host boundary)
