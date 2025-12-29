#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

CONF="${CONF:-pr}"
COUNT="${COUNT:-10}"
export CONF COUNT

"$(dirname "${BASH_SOURCE[0]}")/install_tools.sh"
"$(dirname "${BASH_SOURCE[0]}")/run_sweet.sh"
"$(dirname "${BASH_SOURCE[0]}")/aggregate_results.sh"
"$(dirname "${BASH_SOURCE[0]}")/withmeta.sh"

PR_RESULTS="${PR_RESULTS:-${ARTIFACTS_DIR}/${CONF}.withmeta.results}"
OUTPUT="${OUTPUT:-${ARTIFACTS_DIR}/benchstat.txt}"
export PR_RESULTS OUTPUT

"$(dirname "${BASH_SOURCE[0]}")/benchstat.sh"
