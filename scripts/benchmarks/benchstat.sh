#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

BASELINE="${BASELINE:-$BASELINE_FILE}"
PR_RESULTS="${PR_RESULTS:?set PR_RESULTS to the PR .withmeta.results file}"
OUTPUT="${OUTPUT:-$ARTIFACTS_DIR/benchstat.txt}"

if [ ! -f "$BASELINE" ]; then
  echo "missing baseline file: $BASELINE" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT")"

"$BENCHSTAT_BIN" "$BASELINE" "$PR_RESULTS" > "$OUTPUT"
