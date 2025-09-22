#!/usr/bin/env bash
set -euo pipefail

# Ensure go.mod/go.sum stay tidy without requiring external hooks.
GO=${GO:-go}

# Run in repo root (script lives in scripts/)
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
cd "$REPO_ROOT"

$GO mod tidy

if ! git diff --quiet -- go.mod go.sum; then
  echo "go.mod or go.sum changed; run 'go mod tidy' and commit the result." >&2
  git --no-pager diff -- go.mod go.sum >&2 || true
  exit 1
fi
