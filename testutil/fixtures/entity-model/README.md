# Entity Model Fixtures

This directory now holds the canonical synthetic dataset generated from `docs/schema/entity-model.json`. The goal is to exercise referential integrity, lifecycle transitions, and invariant checks (housing capacity, protocol subject caps, lineage integrity, lifecycle transitions, and protocol coverage) across both SQLite snapshots and the upcoming normalized Postgres driver.

Guidelines:
- Keep fixtures species-agnostic and rely only on fields present in `entity-model.json`.
- Cover every core entity and at least one example of each relationship cardinality (0..1, 1..1, 0..n, 1..n).
- Include lifecycle state transitions that hit initial and terminal states so rules can evaluate edge paths.
- Fixtures are generated as JSON per entity bucket (mirroring the current store layout) via `make entity-model-generate` (invokes `internal/tools/entitymodel/generate -fixtures testutil/fixtures/entity-model/snapshot.json`). Do not hand-edit `snapshot.json`; regenerate from the schema instead.
- Extension slots remain maps by design; payloads are nested under the `core` plugin key so the domain extension containers can validate them on import.

Validation:
- `make lint` regenerates fixtures and fails if they drift from the schema; `make test` loads `snapshot.json` into both the in-memory and SQLite stores and evaluates the built-in invariants (see `internal/core/entity_model_fixtures_test.go`).
- Postgres adapter coverage will reuse the same fixtures once the adapter is implemented.
- Drift guardrails:
  - `make entity-model-generate` (called by `make lint`/`make entity-model-verify`) rewrites `snapshot.json` from `docs/schema/entity-model.json`.
  - `internal/tools/entitymodel/generate/main_test.go` fails if the committed snapshot differs from generator output, ensuring contributors commit the refreshed file alongside schema changes.
