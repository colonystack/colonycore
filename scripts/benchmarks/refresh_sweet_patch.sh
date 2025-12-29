#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/env.sh"

mkdir -p "$BENCH_CACHE_DIR"

ensure_sweet_repo_exists_and_checkout "$SWEET_SRC_DIR" "$SWEET_COMMIT"

rm -rf "$SWEET_WORK_DIR"
cp -a "$SWEET_SRC_DIR" "$SWEET_WORK_DIR"

if ! git -C "$SWEET_WORK_DIR" apply "$PATCH_FILE"; then
  echo "unable to apply ${PATCH_FILE} to ${SWEET_COMMIT}; resolve conflicts before re-diffing" >&2
  exit 1
fi

git -C "$SWEET_WORK_DIR" add -N sweet

tmp_patch="$(mktemp "${PATCH_FILE}.XXXXXX")"
git -C "$SWEET_WORK_DIR" diff --no-color -U1 | sed '/^index /d' > "$tmp_patch"

if [ ! -s "$tmp_patch" ]; then
  echo "generated patch is empty; refusing to overwrite ${PATCH_FILE}" >&2
  rm -f "$tmp_patch"
  exit 1
fi

mv "$tmp_patch" "$PATCH_FILE"
echo "refreshed ${PATCH_FILE} for ${SWEET_COMMIT}"
