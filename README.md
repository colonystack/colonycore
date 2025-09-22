# ColonyCore

ColonyCore is an extensible base module for laboratory colony management. It couples a rules-driven core domain service with pluggable species modules and tooling to validate the project registry.

## Project layout
- `internal/core/` contains the domain model, services, stores, and shared rules engine primitives.
- `plugins/` hosts externally consumable plugins (for example `plugins/frog`) that register species-specific schemas and rules.
- `cmd/registry-check/` provides the CLI used to validate `docs/rfc/registry.yaml` against the expected structure.
- `docs/` captures design history (`docs/adr/`), planning RFCs (`docs/rfc/`), operational annexes (`docs/annex/`), and machine-readable schemas (`docs/schema/`).
- `Makefile` orchestrates common build, lint, and test workflows.

## Getting started
1. Install [Go](https://go.dev/dl/) 1.25 or newer and ensure `$GOPATH/bin` is on your `PATH`.
2. Clone this repository and change into its directory.
3. Run `go test ./...` (or `make test`) to verify your toolchain and dependencies.

## Development workflow
- Build all packages with `make build`.
- Compile the registry validator via `make registry-check`, which outputs `cmd/registry-check/registry-check`.
- Validate the governance registry using `make registry-lint` or by running `go run ./cmd/registry-check --registry docs/rfc/registry.yaml`.
- Refer to `CONTRIBUTING.md` for coding standards, workflow expectations, and pull request guidance.

## Testing
- `make test` runs the race-enabled Go test suite and enforces the configured coverage threshold.
- `make lint` chains formatting checks, vetting, registry validation, and `golangci-lint` (install instructions are in the Makefile). Use `SKIP_GOLANGCI=1 make lint` to bypass the aggregated linter when necessary.
- Additional manual validation steps are described in relevant documents under `docs/`.

## Documentation and support
- Review `CODE_OF_CONDUCT.md` and `CONTRIBUTING.md` before opening issues or pull requests.
- Ownership and escalation paths are tracked in `MAINTAINERS.md` and `SECURITY.md`; the latter also details the coordinated vulnerability disclosure process.
- Long-form decision records and proposals live in `docs/adr/` and `docs/rfc/`. Operational considerations are captured in `docs/annex/`.
- All source code is licensed under the terms of `LICENSE`.
