#!/usr/bin/env bash
set -euo pipefail

MODE="${VERIFY_MODE:-full}"
if [[ "${1:-}" == "--mode" ]]; then
  MODE="${2:-}"
  shift 2
fi
if [[ "$MODE" != "fast" && "$MODE" != "full" ]]; then
  echo "ERROR: invalid mode '$MODE' (expected: fast|full)"
  exit 1
fi

echo "==> gofumpt (check only)"
if ! command -v gofumpt >/dev/null 2>&1; then
  echo "ERROR: gofumpt not found. Install with:"
  echo "  go install mvdan.cc/gofumpt@latest"
  exit 1
fi
mapfile -t GO_FILES < <(git ls-files '*.go')
if [[ ${#GO_FILES[@]} -gt 0 ]]; then
  UNFORMATTED=$(gofumpt -l "${GO_FILES[@]}" 2>/dev/null || true)
  if [[ -n "${UNFORMATTED}" ]]; then
    echo "ERROR: gofumpt required for:"
    echo "${UNFORMATTED}"
    exit 1
  fi
fi

echo "==> go test"
if [[ "$MODE" == "full" ]]; then
  go test -race ./... -v
else
  go test ./... -v
fi
