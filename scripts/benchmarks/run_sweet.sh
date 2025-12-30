#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

CONF="${CONF:?set CONF to identify the run (baseline or pr)}"
COUNT="${COUNT:-10}"

mkdir -p "$RESULTS_DIR" "$ASSETS_DIR" "$BENCH_CACHE_DIR"

"$(dirname "${BASH_SOURCE[0]}")/prepare_sweet.sh"

GOROOT_PATH="$(go env GOROOT)"
CONFIG_FILE="${BENCH_CACHE_DIR}/config-${CONF}.toml"

cat > "$CONFIG_FILE" <<EOF
[[config]]
  name = "${CONF}"
  goroot = "${GOROOT_PATH}"
  envexec = ["GOMAXPROCS=1"]
EOF

export COLONYCORE_REPO_ROOT="${COLONYCORE_REPO_ROOT:-$ROOT_DIR}"

"$SWEET_BIN" run \
  -bench-dir "${SWEET_WORK_DIR}/sweet/benchmarks" \
  -assets-dir "$ASSETS_DIR" \
  -results "$RESULTS_DIR" \
  -work-dir "${BENCH_CACHE_DIR}/work" \
  -count "$COUNT" \
  -run colonycore \
  "$CONFIG_FILE"
