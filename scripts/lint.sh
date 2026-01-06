#!/usr/bin/env bash
set -euo pipefail

echo "Running golangci-lint..."
if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint not found; install it with 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'"
  exit 1
fi

golangci-lint run ./...
