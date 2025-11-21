# Entity Model Fixtures

This directory will hold synthetic datasets derived from `docs/schema/entity-model.json`. The goal is to exercise referential integrity, lifecycle transitions, and invariant checks (housing capacity, protocol subject caps, and future ADR-0003 invariants) across both SQLite snapshots and the upcoming normalized Postgres driver.

Guidelines:
- Keep fixtures species-agnostic and rely only on fields present in `entity-model.json`.
- Cover every core entity and at least one example of each relationship cardinality (0..1, 1..1, 0..n, 1..n).
- Include lifecycle state transitions that hit initial and terminal states so rules can evaluate edge paths.
- Store fixtures as JSON per entity bucket (mirroring the current store layout) until generators emit canonical formats.

Validation:
- Conformance tests will load these fixtures into both SQLite and Postgres adapters, then run rule checks. Extend this document with any generator instructions once those land.
