#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

mkdir -p "$BIN_DIR"

GOBIN="$BIN_DIR" \
  GOCACHE="$ROOT_DIR/.cache/go-build" \
  go install "golang.org/x/perf/cmd/benchstat@${BENCHSTAT_VERSION}"
