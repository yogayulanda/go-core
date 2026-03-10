#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

lint_bin="${GOLANGCI_LINT:-}"
if [[ -z "$lint_bin" ]]; then
  if command -v golangci-lint >/dev/null 2>&1; then
    lint_bin="golangci-lint"
  else
    lint_bin="$(go env GOPATH)/bin/golangci-lint"
  fi
fi

if [[ ! -x "$(command -v "$lint_bin" 2>/dev/null || true)" ]] && [[ ! -x "$lint_bin" ]]; then
  echo "golangci-lint not found. install with:"
  echo "go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
  exit 1
fi

echo "[quality] go test ./..."
go test ./...

echo "[quality] go test -race ./..."
go test -race ./...

echo "[quality] go vet ./..."
go vet ./...

echo "[quality] golangci-lint run"
"$lint_bin" run

echo "[security] golangci-lint run -E gosec --tests=false"
"$lint_bin" run -E gosec --tests=false

echo "quality gate passed"
