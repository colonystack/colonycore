GOCACHE ?= $(CURDIR)/.cache/go-build
COVERFILE ?= coverage.out
COVERMODE ?= atomic
GOLANGCI_VERSION ?= v2.5.0
GOLANGCI_PKG := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
BIN_DIR ?= $(CURDIR)/bin
GOLANGCI_CACHE ?= $(CURDIR)/.cache/golangci-lint
COVER_THRESHOLD ?= 90.0

.PHONY: all build lint go-test test registry-check fmt-check vet registry-lint golangci golangci-install python-lint r-lint go-lint

all: build

build:
	GOCACHE=$(GOCACHE) go build ./...

registry-check:
	GOCACHE=$(GOCACHE) go build -o cmd/registry-check/registry-check ./cmd/registry-check

lint:
	@$(MAKE) --no-print-directory go-lint
	@$(MAKE) --no-print-directory python-lint
	@$(MAKE) --no-print-directory r-lint
	@echo "Lint suite finished successfully"

go-lint:
	@echo "==> Go lint"
	@$(MAKE) --no-print-directory fmt-check
	@$(MAKE) --no-print-directory vet
	@$(MAKE) --no-print-directory registry-lint
	@$(MAKE) --no-print-directory golangci
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
		if ! command -v golangci-lint >/dev/null 2>&1; then \
			echo "golangci-lint not found. Installing $(GOLANGCI_PKG) into $(BIN_DIR)"; \
			$(MAKE) golangci-install; \
		fi; \
		mkdir -p $(GOLANGCI_CACHE); \
		PATH="$(BIN_DIR):$$PATH" \
		GOCACHE="$(GOCACHE)" \
		GOLANGCI_LINT_CACHE="$(GOLANGCI_CACHE)" \
		golangci-lint run --timeout=30m --fix ./...; \
	fi

golangci-install:
	@mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install $(GOLANGCI_PKG)

python-lint:
	@echo "==> Python lint"
	@python -m ruff check --quiet clients/python || ( \
		echo "ruff is required. Install it via 'pip install ruff' or rerun pre-commit to bootstrap it." >&2; \
		exit 1 )
	@echo "Python lint: OK"

r-lint:
	@echo "==> R lint"
	@python scripts/run_lintr.py
	@echo "R lint: OK"

go-test:
	GOCACHE=$(GOCACHE) go test -race -covermode=$(COVERMODE) -coverprofile=$(COVERFILE) ./...
	@GOCACHE=$(GOCACHE) go tool cover -func=$(COVERFILE) > coverage.summary
	@awk '/^total:/ { if ($$3+0 < $(COVER_THRESHOLD)) { printf "Coverage %.1f%% is below threshold $(COVER_THRESHOLD)%%\n", $$3+0; exit 1 } }' coverage.summary
	@echo "Coverage check passed (>= $(COVER_THRESHOLD)% )"

test: go-test
