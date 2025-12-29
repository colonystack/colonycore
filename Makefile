GOCACHE ?= $(CURDIR)/.cache/go-build
COVERFILE ?= coverage.out
COVERMODE ?= atomic
GOLANGCI_VERSION ?= v2.7.2
BIN_DIR ?= $(CURDIR)/.cache/bin
GOLANGCI_CACHE ?= $(CURDIR)/.cache/golangci-lint
COVER_THRESHOLD ?= 90.0
SWEET_VERSION ?= v0.0.0-20251208221949-523919e4e4f2
SWEET_COMMIT ?= 523919e4e4f284a0c060e6e5e5ff7f6f521fa2ed
BENCHSTAT_VERSION ?= v0.0.0-20251208221838-04cf7a2dca90
BENCH_CONF ?= pr
BENCH_COUNT ?= 10
GOPATH_BIN := $(shell go env GOPATH)/bin
GOLANGCI_BIN := $(GOPATH_BIN)/golangci-lint
GOLANGCI_VERSION_PLAIN := $(patsubst v%,%,$(GOLANGCI_VERSION))
MODULE := $(shell go list -m)
IMPORT_BOSS_BIN := $(GOPATH_BIN)/import-boss
SCHEMASPY_IMAGE ?= schemaspy/schemaspy:7.0.2
SCHEMASPY_PLATFORM ?=
SCHEMASPY_TMP := $(CURDIR)/.cache/schemaspy/entitymodel-erd
SCHEMASPY_SVG_OUT := $(CURDIR)/docs/annex/entity-model-erd.svg
SCHEMASPY_DOT_OUT := $(CURDIR)/docs/annex/entity-model-erd.dot
SCHEMASPY_PG_IMAGE ?= postgres:16-alpine
SCHEMASPY_PG_PLATFORM ?= linux/amd64
SCHEMASPY_PG_CONTAINER ?= entitymodel-erd-pg
SCHEMASPY_PG_DB ?= entitymodel
SCHEMASPY_PG_USER ?= postgres
SCHEMASPY_PG_PASSWORD ?= postgres
SCHEMASPY_PG_TIMEOUT ?= 60

.PHONY: all build lint go-test test registry-check fmt-check vet registry-lint golangci golangci-install python-lint r-lint go-lint import-boss import-boss-install entity-model-validate entity-model-generate entity-model-verify entity-model-erd entity-model-diff entity-model-diff-update api-snapshots list-docker-images validate-any-usage benchmarks-run benchmarks-aggregate benchmarks-compare benchmarks-ci

all: build

list-docker-images:
	@echo "$(SCHEMASPY_IMAGE)"
	@echo "$(SCHEMASPY_PG_IMAGE)"

build:
	GOCACHE=$(GOCACHE) go build ./...

registry-check:
	GOCACHE=$(GOCACHE) go build -o cmd/registry-check/registry-check ./cmd/registry-check

lint:
	@$(MAKE) --no-print-directory entity-model-verify
	@$(MAKE) --no-print-directory entity-model-diff
	@$(MAKE) --no-print-directory api-snapshots
	@$(MAKE) --no-print-directory go-lint
	@$(MAKE) --no-print-directory validate-plugin-patterns
	@$(MAKE) --no-print-directory validate-any-usage
	@$(MAKE) --no-print-directory python-lint
	@$(MAKE) --no-print-directory r-lint
	@echo "Lint suite finished successfully"

validate-plugin-patterns:
	@echo "==> Validating plugin hexagonal architecture patterns"
	@for plugin_dir in plugins/*/; do \
		if [ -d "$$plugin_dir" ] && find "$$plugin_dir" -name '*.go' ! -name '*_test.go' | grep -q .; then \
			echo "Validating plugin: $$plugin_dir"; \
			go run scripts/validate_plugin_patterns.go "$$plugin_dir"; \
		fi; \
	done
	@echo "validate-plugin-patterns: OK"

validate-any-usage:
	@echo "==> Validating any usage"
	@GOCACHE=$(GOCACHE) go run ./scripts/validate_any_usage
	@echo "validate-any-usage: OK"

benchmarks-run:
	@CONF=$(BENCH_CONF) COUNT=$(BENCH_COUNT) SWEET_VERSION=$(SWEET_VERSION) SWEET_COMMIT=$(SWEET_COMMIT) BENCHSTAT_VERSION=$(BENCHSTAT_VERSION) scripts/benchmarks/run_sweet.sh

benchmarks-aggregate:
	@CONF=$(BENCH_CONF) SWEET_VERSION=$(SWEET_VERSION) SWEET_COMMIT=$(SWEET_COMMIT) BENCHSTAT_VERSION=$(BENCHSTAT_VERSION) scripts/benchmarks/aggregate_results.sh
	@CONF=$(BENCH_CONF) SWEET_VERSION=$(SWEET_VERSION) SWEET_COMMIT=$(SWEET_COMMIT) BENCHSTAT_VERSION=$(BENCHSTAT_VERSION) scripts/benchmarks/withmeta.sh

benchmarks-compare:
	@PR_RESULTS=$(CURDIR)/benchmarks/artifacts/$(BENCH_CONF).withmeta.results SWEET_VERSION=$(SWEET_VERSION) SWEET_COMMIT=$(SWEET_COMMIT) BENCHSTAT_VERSION=$(BENCHSTAT_VERSION) scripts/benchmarks/benchstat.sh

benchmarks-ci:
	@CONF=$(BENCH_CONF) COUNT=$(BENCH_COUNT) SWEET_VERSION=$(SWEET_VERSION) SWEET_COMMIT=$(SWEET_COMMIT) BENCHSTAT_VERSION=$(BENCHSTAT_VERSION) scripts/benchmarks/ci.sh

api-snapshots:
	@echo "==> api snapshots"
	@GOCACHE=$(GOCACHE) go test ./pkg/pluginapi -run TestGeneratePluginAPISnapshot -update
	@GOCACHE=$(GOCACHE) go test ./pkg/datasetapi -run TestGenerateDatasetAPISnapshot -update
	@echo "api-snapshots: OK"

go-lint:
	@echo "==> Go lint"
	@$(MAKE) --no-print-directory fmt-check
	@$(MAKE) --no-print-directory vet
	@$(MAKE) --no-print-directory registry-lint
	@$(MAKE) --no-print-directory golangci
	@$(MAKE) --no-print-directory import-boss
	@echo "Go lint: OK"

fmt-check:
	@files="$$(find . -path './.git' -prune -o -path './.cache' -prune -o -name '*.go' -print)"; \
	if [ -n "$$files" ]; then \
		out="$$(gofmt -l -s $$files)"; \
		if [ -n "$$out" ]; then \
			echo 'gofmt required for:'; \
			echo "$$out"; \
			exit 1; \
		fi; \
	fi

vet:
	GOCACHE=$(GOCACHE) go vet ./...

registry-lint: registry-check
	GOCACHE=$(GOCACHE) ./cmd/registry-check/registry-check -registry docs/rfc/registry.yaml

golangci:
	@set -e; \
	if [ -n "$(SKIP_GOLANGCI)" ]; then \
		echo "Skipping golangci-lint because SKIP_GOLANGCI is set"; \
	else \
		need_install=0; \
		if [ ! -x "$(GOLANGCI_BIN)" ]; then \
			need_install=1; \
		else \
			installed_version=$$($(GOLANGCI_BIN) version 2>/dev/null | sed -n 's/.*version \([0-9][0-9.]*\).*/\1/p'); \
			if [ -z "$$installed_version" ]; then \
				echo "Could not determine golangci-lint version. Reinstalling"; \
				need_install=1; \
			elif [ "$$installed_version" != "$(GOLANGCI_VERSION_PLAIN)" ]; then \
				echo "golangci-lint version $$installed_version (want $(GOLANGCI_VERSION_PLAIN)). Reinstalling"; \
				need_install=1; \
			fi; \
		fi; \
		if [ $$need_install -eq 1 ]; then \
			$(MAKE) golangci-install; \
		fi; \
		mkdir -p $(GOLANGCI_CACHE); \
		tmpfile=$$(mktemp); \
		if ! GOCACHE="$(GOCACHE)" GOLANGCI_LINT_CACHE="$(GOLANGCI_CACHE)" $(GOLANGCI_BIN) run --timeout=30m --fix ./... >$$tmpfile 2>&1; then \
			cat $$tmpfile; \
			rm -f $$tmpfile; \
			exit 1; \
		else \
			echo "golangci-lint: OK"; \
			rm -f $$tmpfile; \
		fi; \
	fi

golangci-install:
	@echo "Installing golangci-lint $(GOLANGCI_VERSION) to $(GOPATH_BIN) using official script"
	@# Suppress all output from the install script but fail loudly if it errors
	@if ! curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
		| sh -s -- -b $(GOPATH_BIN) $(GOLANGCI_VERSION) >/dev/null 2>&1; then \
		echo "golangci-lint install failed" >&2; \
		exit 1; \
	fi

import-boss:
	@set -e; \
	if [ ! -x "$(IMPORT_BOSS_BIN)" ]; then \
		$(MAKE) --no-print-directory import-boss-install; \
	fi; \
	pkgs=$$(find . -path './.cache' -prune -o -name '.import-restrictions' -print | while read -r file; do \
		dir=$$(dirname "$$file"); \
		if find "$$dir" -maxdepth 1 -name '*.go' -print -quit | grep -q .; then \
			rel=$${dir#./}; \
			if [ "$$rel" = "" ]; then \
				echo "$(MODULE)"; \
			else \
				echo "$(MODULE)/$$rel"; \
			fi; \
		fi; \
	done | sort -u); \
	if [ -z "$$pkgs" ]; then \
		echo "import-boss: no packages to verify"; \
	else \
		inputs=$$(printf '%s\n' $$pkgs | paste -sd, -); \
		echo "==> import-boss"; \
		mkdir -p $(GOCACHE); \
		GOCACHE="$(GOCACHE)" $(IMPORT_BOSS_BIN) --alsologtostderr=false --logtostderr=false --stderrthreshold=ERROR --verify-only --input-dirs $$inputs; \
		echo "import-boss: OK"; \
	fi

import-boss-install:
	@echo "Installing import-boss via go get"
	@mkdir -p $(dir $(IMPORT_BOSS_BIN))
	@GOCACHE=$(GOCACHE) go get k8s.io/gengo/examples/import-boss
	@GOCACHE=$(GOCACHE) go install k8s.io/gengo/examples/import-boss
	@GOCACHE=$(GOCACHE) go mod tidy

python-lint:
	@echo "==> Python lint"
	@python -m ruff check --quiet clients/python || ( \
		echo "ruff is required. Install it via 'pip install ruff' or rerun pre-commit to bootstrap it." >&2; \
		exit 1 )
	@echo "Python lint: OK"

r-lint:
	@echo "==> R lint"
	@python scripts/run_lintr.py && echo "R lint: OK" || (status=$$?; if [ $$status -eq 0 ]; then echo "R lint: OK"; else exit $$status; fi)

entity-model-validate:
	@echo "==> entity-model validate"
	@GOCACHE=$(GOCACHE) go run ./internal/tools/entitymodel/validate docs/schema/entity-model.json

entity-model-generate:
	@echo "==> entity-model generate"
	@GOCACHE=$(GOCACHE) go run ./internal/tools/entitymodel/generate -schema docs/schema/entity-model.json -out pkg/domain/entitymodel/model_gen.go -openapi docs/schema/openapi/entity-model.yaml -sql-postgres docs/schema/sql/postgres.sql -sql-sqlite docs/schema/sql/sqlite.sql -plugin-contract docs/annex/plugin-contract.md -fixtures testutil/fixtures/entity-model/snapshot.json -pluginapi-constants pkg/pluginapi/entity_states_gen.go -datasetapi-constants pkg/datasetapi/entity_states_gen.go
	@$(MAKE) --no-print-directory entity-model-erd

entity-model-verify: entity-model-validate entity-model-generate
	@echo "==> entity-model verify (validate + generation)"

entity-model-diff:
	@echo "==> entity-model diff"
	@GOCACHE=$(GOCACHE) go run ./internal/tools/entitymodel/diff -schema docs/schema/entity-model.json -fingerprint docs/schema/entity-model.fingerprint.json

entity-model-diff-update:
	@echo "==> entity-model diff (write)"
	@GOCACHE=$(GOCACHE) go run ./internal/tools/entitymodel/diff -schema docs/schema/entity-model.json -fingerprint docs/schema/entity-model.fingerprint.json -write

entity-model-erd:
	@echo "==> entity-model erd (SchemaSpy via generated Postgres DDL)"
	@rm -rf $(SCHEMASPY_TMP)
	@mkdir -p $(SCHEMASPY_TMP) $(dir $(SCHEMASPY_SVG_OUT))
	@chmod 777 $(SCHEMASPY_TMP)
	@docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1 || true
	@docker run --rm -d --name $(SCHEMASPY_PG_CONTAINER) --platform $(SCHEMASPY_PG_PLATFORM) -e POSTGRES_PASSWORD=$(SCHEMASPY_PG_PASSWORD) -e POSTGRES_DB=$(SCHEMASPY_PG_DB) $(SCHEMASPY_PG_IMAGE) >/dev/null 2>&1 || { echo "Failed to start postgres container"; exit 1; }
	@printf "waiting for postgres"
	@timeout=$(SCHEMASPY_PG_TIMEOUT); elapsed=0; \
	until docker exec $(SCHEMASPY_PG_CONTAINER) pg_isready -U $(SCHEMASPY_PG_USER) -d $(SCHEMASPY_PG_DB) >/dev/null 2>&1; do \
		docker ps --filter "name=$(SCHEMASPY_PG_CONTAINER)" --filter "status=running" --format '{{.Names}}' | grep -q "^$(SCHEMASPY_PG_CONTAINER)$$" || { echo " FAILED (container stopped)"; docker logs $(SCHEMASPY_PG_CONTAINER) 2>&1; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }; \
		[ $$elapsed -lt $$timeout ] || { echo " TIMEOUT"; docker logs $(SCHEMASPY_PG_CONTAINER) 2>&1; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }; \
		printf "."; sleep 1; elapsed=$$((elapsed + 1)); \
	done
	@timeout=$(SCHEMASPY_PG_TIMEOUT); elapsed=0; \
	until docker exec $(SCHEMASPY_PG_CONTAINER) sh -c "psql -X -U $(SCHEMASPY_PG_USER) -d postgres -tc \"SELECT 1 FROM pg_database WHERE datname='$(SCHEMASPY_PG_DB)';\" | grep -q 1 || createdb -U $(SCHEMASPY_PG_USER) $(SCHEMASPY_PG_DB)" >/dev/null 2>&1; do \
		docker ps --filter "name=$(SCHEMASPY_PG_CONTAINER)" --filter "status=running" --format '{{.Names}}' | grep -q "^$(SCHEMASPY_PG_CONTAINER)$$" || { echo " FAILED (container stopped)"; docker logs $(SCHEMASPY_PG_CONTAINER) 2>&1; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }; \
		[ $$elapsed -lt $$timeout ] || { echo " TIMEOUT"; docker logs $(SCHEMASPY_PG_CONTAINER) 2>&1; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }; \
		printf "."; sleep 3; elapsed=$$((elapsed + 3)); \
	done; echo "OK"
	@docker exec -i $(SCHEMASPY_PG_CONTAINER) psql -X -v ON_ERROR_STOP=1 -1 -U $(SCHEMASPY_PG_USER) -d $(SCHEMASPY_PG_DB) < docs/schema/sql/postgres.sql >/dev/null 2>&1 || { echo "Schema load failed"; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }
	@docker run --rm $(if $(SCHEMASPY_PLATFORM),--platform $(SCHEMASPY_PLATFORM),) -v "$(SCHEMASPY_TMP)":/output --network container:$(SCHEMASPY_PG_CONTAINER) $(SCHEMASPY_IMAGE) -t pgsql11 -db $(SCHEMASPY_PG_DB) -host localhost -port 5432 -s public -u $(SCHEMASPY_PG_USER) -p $(SCHEMASPY_PG_PASSWORD) -dbthreads 1 -hq -imageformat svg >/dev/null 2>&1 || { echo "SchemaSpy failed"; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }
	@cp "$(SCHEMASPY_TMP)/diagrams/summary/relationships.real.large.svg" "$(SCHEMASPY_SVG_OUT)" || { echo "Failed to copy SVG"; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }
	@cp "$(SCHEMASPY_TMP)/diagrams/summary/relationships.real.large.dot" "$(SCHEMASPY_DOT_OUT)" || { echo "Failed to copy DOT"; docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1; exit 1; }
	@docker rm -f $(SCHEMASPY_PG_CONTAINER) >/dev/null 2>&1 || true
	@echo "SchemaSpy ERD written to $(SCHEMASPY_SVG_OUT) (full report in $(SCHEMASPY_TMP))"
	@echo "SchemaSpy Graphviz DOT written to $(SCHEMASPY_DOT_OUT) (full report in $(SCHEMASPY_TMP))"


go-test:
	@GOCACHE=$(GOCACHE) go test -race -covermode=$(COVERMODE) -coverprofile=$(COVERFILE) ./...
	@GOCACHE=$(GOCACHE) go tool cover -func=$(COVERFILE) > coverage.summary
	@awk '/^total:/ { if ($$3+0 < $(COVER_THRESHOLD)) { printf "Coverage %.1f%% is below threshold $(COVER_THRESHOLD)%%\n", $$3+0; exit 1 } }' coverage.summary
	@echo "Coverage check passed (>= $(COVER_THRESHOLD)% )"

test: go-test
