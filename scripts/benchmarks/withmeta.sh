#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

CONF="${CONF:?set CONF to identify the run (baseline or pr)}"

INPUT="${ARTIFACTS_DIR}/${CONF}.results"
OUTPUT="${ARTIFACTS_DIR}/${CONF}.withmeta.results"

if [ ! -f "$INPUT" ]; then
  echo "missing results file: $INPUT" >&2
  exit 1
fi

mkdir -p "$ARTIFACTS_DIR"

COMMIT_SHA="${GIT_SHA:-$(git -C "$ROOT_DIR" rev-parse HEAD)}"
GO_VERSION="$(go version)"
RUNNER_ID="${RUNNER_ID:-$(hostname)}"

{
  echo "commit: ${COMMIT_SHA}"
  echo "go: ${GO_VERSION}"
  echo "sweet: ${SWEET_VERSION}"
  echo "benchstat: ${BENCHSTAT_VERSION}"
  echo "runner: ${RUNNER_ID}"
  cat "$INPUT"
} > "$OUTPUT"
