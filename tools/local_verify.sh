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

echo "==> local verify mode: $MODE"

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

echo "==> golangci-lint"
if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "ERROR: golangci-lint not found. Install with:"
  echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"
  exit 1
fi
golangci-lint run ./...

if [[ -f tools/releasetag/go.mod ]]; then
  echo "==> golangci-lint (tools/releasetag)"
  (
    cd tools/releasetag
    GOWORK=off golangci-lint run ./...
  )
fi

if [[ "$MODE" == "full" ]]; then
  echo "==> shellcheck"
  if ! command -v shellcheck >/dev/null 2>&1; then
    echo "ERROR: shellcheck not found. Install with:"
    echo "  choco install shellcheck"
    exit 1
  fi
  mapfile -t SH_FILES < <(git ls-files '*.sh' | sort -u)
  if [[ ${#SH_FILES[@]} -gt 0 ]]; then
    shellcheck "${SH_FILES[@]}"
  fi

  echo "==> markdownlint"
  mapfile -t MD_FILES < <(git ls-files '*.md')
  if [[ ${#MD_FILES[@]} -gt 0 ]]; then
    if command -v markdownlint-cli2 >/dev/null 2>&1; then
      markdownlint-cli2 "${MD_FILES[@]}"
    elif command -v npx >/dev/null 2>&1; then
      npx --yes markdownlint-cli2 "${MD_FILES[@]}"
    else
      echo "ERROR: markdownlint-cli2 not found. Install with:"
      echo "  npm install -g markdownlint-cli2"
      exit 1
    fi
  fi

  echo "==> actionlint"
  if ! command -v actionlint >/dev/null 2>&1; then
    echo "ERROR: actionlint not found. Install with:"
    echo "  go install github.com/rhysd/actionlint/cmd/actionlint@latest"
    exit 1
  fi
  actionlint
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
  echo "==> go mod verify"
  go mod verify

  echo "==> go vet"
  go vet ./...
  if [[ -f tools/releasetag/go.mod ]]; then
    echo "==> go vet (tools/releasetag)"
    go -C tools/releasetag vet ./...
  fi

  echo "==> go build"
  go build ./...
  if [[ -f tools/releasetag/go.mod ]]; then
    echo "==> go build (tools/releasetag)"
    go -C tools/releasetag build ./...
  fi

  go test -race ./... -v
else
  go test ./... -v
fi
