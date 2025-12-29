#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

SWEET_VERSION="${SWEET_VERSION:-v0.0.0-20251208221949-523919e4e4f2}"
SWEET_COMMIT="${SWEET_COMMIT:-523919e4e4f284a0c060e6e5e5ff7f6f521fa2ed}"
BENCHSTAT_VERSION="${BENCHSTAT_VERSION:-v0.0.0-20251208221838-04cf7a2dca90}"
BENCHMARKS_REPO="${BENCHMARKS_REPO:-https://go.googlesource.com/benchmarks}"

BENCH_CACHE_DIR="${BENCH_CACHE_DIR:-$ROOT_DIR/.cache/benchmarks}"
SWEET_SRC_DIR="${SWEET_SRC_DIR:-$BENCH_CACHE_DIR/sweet-src-${SWEET_COMMIT}}"
SWEET_WORK_DIR="${SWEET_WORK_DIR:-$BENCH_CACHE_DIR/sweet-work}"

BIN_DIR="${BIN_DIR:-$ROOT_DIR/.cache/bin}"
SWEET_BIN="${SWEET_BIN:-$BIN_DIR/sweet}"
BENCHSTAT_BIN="${BENCHSTAT_BIN:-$BIN_DIR/benchstat}"

RESULTS_DIR="${RESULTS_DIR:-$ROOT_DIR/benchmarks/results}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-$ROOT_DIR/benchmarks/artifacts}"
ASSETS_DIR="${ASSETS_DIR:-$ROOT_DIR/benchmarks/assets-empty}"
BASELINE_FILE="${BASELINE_FILE:-$ROOT_DIR/internal/ci/benchmarks/baseline.withmeta.results}"

PATCH_FILE="${PATCH_FILE:-$ROOT_DIR/scripts/benchmarks/sweet_colonycore.patch}"
SWEET_PATCH_TARGET="${SWEET_PATCH_TARGET:-sweet}"

ensure_sweet_repo_exists_and_checkout() {
  local repo_dir="$1"
  local commit="$2"
  local repo_url="${3:-$BENCHMARKS_REPO}"

  if [ ! -d "$repo_dir/.git" ]; then
    git clone "$repo_url" "$repo_dir" || {
      echo "Error: failed to clone benchmarks repo into $repo_dir" >&2
      exit 1
    }
  fi

  if ! git -C "$repo_dir" cat-file -e "${commit}^{commit}" 2>/dev/null; then
    git -C "$repo_dir" fetch origin || {
      echo "Error: failed to fetch benchmarks repo in $repo_dir" >&2
      exit 1
    }
  fi
  git -C "$repo_dir" checkout "$commit" || {
    echo "Error: failed to checkout $commit in $repo_dir" >&2
    exit 1
  }
}
