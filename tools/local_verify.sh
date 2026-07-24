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

export GOWORK=off

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
  echo "==> go generate (drift check)"
  before_mod=$(mktemp)
  after_mod=$(mktemp)
  before_untracked=$(mktemp)
  after_untracked=$(mktemp)
  trap 'rm -f "$before_mod" "$after_mod" "$before_untracked" "$after_untracked"' RETURN

  git diff --name-only -- >"${before_mod}"
  git ls-files --others --exclude-standard >"${before_untracked}"

  go generate ./...

  git diff --name-only -- >"${after_mod}"
  git ls-files --others --exclude-standard >"${after_untracked}"

  new_mod=$(comm -13 <(sort "${before_mod}") <(sort "${after_mod}") || true)
  new_untracked=$(comm -13 <(sort "${before_untracked}") <(sort "${after_untracked}") || true)
  if [[ -n "${new_mod}" || -n "${new_untracked}" ]]; then
    echo "ERROR: go generate introduced new drift."
    if [[ -n "${new_mod}" ]]; then
      echo "Newly modified tracked files:"
      echo "${new_mod}"
    fi
    if [[ -n "${new_untracked}" ]]; then
      echo "Newly created untracked files:"
      echo "${new_untracked}"
    fi
    exit 1
  fi
fi

if [[ "$MODE" == "full" ]]; then
  go test -race ./... -v
else
  go test ./... -v
fi
