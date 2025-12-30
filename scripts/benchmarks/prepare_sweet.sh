#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

mkdir -p "$BENCH_CACHE_DIR" "$BIN_DIR"

ensure_sweet_repo_exists_and_checkout "$SWEET_SRC_DIR" "$SWEET_COMMIT"

rm -rf "$SWEET_WORK_DIR"
cp -a "$SWEET_SRC_DIR" "$SWEET_WORK_DIR"

if ! git -C "$SWEET_WORK_DIR" apply --check "$PATCH_FILE"; then
  echo "sweet patch no longer applies cleanly to ${SWEET_COMMIT}; refresh ${PATCH_FILE}" >&2
  exit 1
fi

git -C "$SWEET_WORK_DIR" apply "$PATCH_FILE"
verify_sweet_overlay_matches_patch "$SWEET_WORK_DIR"
apply_sweet_overlays "$SWEET_WORK_DIR"

(
  cd "$SWEET_WORK_DIR"
  GOCACHE="$ROOT_DIR/.cache/go-build" \
    go build -o "$SWEET_BIN" ./sweet/cmd/sweet
)
