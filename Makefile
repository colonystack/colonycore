GOCACHE ?= $(CURDIR)/.cache/go-build
COVERFILE ?= coverage.out
COVERMODE ?= atomic
GOLANGCI_VERSION ?=
GOLANGCI_PKG := github.com/golangci/golangci-lint/cmd/golangci-lint@$(or $(GOLANGCI_VERSION),latest)
BIN_DIR ?= $(CURDIR)/bin
GOLANGCI_CACHE ?= $(CURDIR)/.cache/golangci-lint
COVER_THRESHOLD ?= 90.0

.PHONY: all build lint go-test test registry-check fmt-check vet registry-lint golangci golangci-install

all: build

build:
	GOCACHE=$(GOCACHE) go build ./...

registry-check:
	GOCACHE=$(GOCACHE) go build ./cmd/registry-check

lint: fmt-check vet registry-lint golangci

fmt-check:
	@files="$$(find . -path './.git' -prune -o -path './.cache' -prune -o -name '*.go' -print)"; \
	if [ -n "$$files" ]; then \
		out="$$(gofmt -l $$files)"; \
		if [ -n "$$out" ]; then \
			echo 'gofmt required for:'; \
			echo "$$out"; \
			exit 1; \
		fi; \
	fi

vet:
	GOCACHE=$(GOCACHE) go vet ./...

registry-lint: registry-check
	GOCACHE=$(GOCACHE) ./registry-check -registry docs/rfc/registry.yaml

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

go-test:
	GOCACHE=$(GOCACHE) go test -race -covermode=$(COVERMODE) -coverprofile=$(COVERFILE) ./...
	@GOCACHE=$(GOCACHE) go tool cover -func=$(COVERFILE) > coverage.summary
	@awk '/^total:/ { if ($$3+0 < $(COVER_THRESHOLD)) { printf "Coverage %.1f%% is below threshold $(COVER_THRESHOLD)%%\n", $$3+0; exit 1 } }' coverage.summary
	@echo "Coverage check passed (>= $(COVER_THRESHOLD)% )"

test: go-test
