# Annex 0009: Dataset Service Dependency Approvals

- Status: Accepted
- Owners: Core Maintainers
- Last Updated: 2026-03-17
- Related: RFC-0001 §9, ADR-0006, issue #28

## Purpose

Record the explicit human approval required by `AGENTS.md` before future
dataset service hardening phases add new Go dependencies.

This annex is a scope gate only. It does not by itself introduce code,
handlers, endpoints, schema changes, or public API changes.

## Approval Record

Approval was granted in the active implementation session on 2026-03-17 with
the exact statement:

> Approval for this scope is given.

That approval covers the dependency request described in issue #28 for the
dataset service production-hardening plan.

## Approved Dependencies

The following Go dependencies are approved for the later implementation phases
named in issue #28:

1. `github.com/prometheus/client_golang`
   Prometheus metrics client for Go. Intended scope: service instrumentation and
   a `/metrics` endpoint for dataset service runtime metrics.
2. `github.com/swaggo/swag`
   Swagger/OpenAPI annotation processor for generating a developer-facing spec
   from Go handlers.
3. `github.com/swaggo/http-swagger`
   HTTP handler for serving Swagger UI backed by generated spec assets.

## Phase Gates

- Phase 1 may add `github.com/prometheus/client_golang` under this recorded
  approval. If a different metrics dependency is preferred, that alternative
  must be documented before Phase 1 starts.
- Phase 8 may add `github.com/swaggo/swag` and
  `github.com/swaggo/http-swagger` under this recorded approval. If a different
  interactive API docs stack is preferred, that alternative must be documented
  before Phase 8 starts.

## Architectural Constraints

- ADR-0006 remains the governing observability baseline. Structured events stay
  canonical even if direct Prometheus instrumentation is added later.
- Any future Prometheus metrics must complement, not replace, the structured
  event contract described in `internal/observability` and
  `observability/README.md`.
- Any future Swagger generation or UI wiring must remain developer-facing and
  must not silently redefine the dataset wire contract; the canonical checked-in
  schema remains `docs/schema/dataset-service.openapi.yaml` until a governing
  RFC/ADR says otherwise.
- Later implementation PRs must still satisfy the normal repo requirements:
  focused scope, rationale for the added dependency, tests, `make lint`,
  `make test`, and any relevant generated artifact verification.

## Risks Carried Forward

- `github.com/prometheus/client_golang` increases the compiled binary size and
  expands the runtime HTTP surface when `/metrics` is enabled.
- `swaggo/swag` generated artifacts can drift from handler annotations if the
  generation workflow is not kept explicit and enforced in CI.
- `swaggo/http-swagger` adds a developer-facing UI surface that must stay out of
  production-critical control paths and access reviews.

## References

- `AGENTS.md`
- `docs/rfc/0001-colonycore-base-module.md`
- `docs/adr/0006-observability-architecture.md`
- `docs/schema/dataset-service.openapi.yaml`
