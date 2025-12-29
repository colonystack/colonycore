#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

CONF="${CONF:?set CONF to identify the run (baseline or pr)}"

mkdir -p "$ARTIFACTS_DIR"

mapfile -d '' files < <(find "$RESULTS_DIR" -type f -name "${CONF}.results" -print0 | sort -z)
if [ "${#files[@]}" -eq 0 ]; then
  echo "no results found for ${CONF} in ${RESULTS_DIR}" >&2
  exit 1
fi

printf '%s\0' "${files[@]}" | xargs -0 cat > "${ARTIFACTS_DIR}/${CONF}.results"
