GOCACHE ?= $(CURDIR)/.cache/go-build
COVERFILE ?= coverage.out
COVERMODE ?= atomic
GOLANGCI_VERSION ?= v2.5.0
BIN_DIR ?= $(CURDIR)/.cache/bin
GOLANGCI_CACHE ?= $(CURDIR)/.cache/golangci-lint
COVER_THRESHOLD ?= 90.0
GOPATH_BIN := $(shell go env GOPATH)/bin
GOLANGCI_BIN := $(GOPATH_BIN)/golangci-lint
GOLANGCI_VERSION_PLAIN := $(patsubst v%,%,$(GOLANGCI_VERSION))

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
