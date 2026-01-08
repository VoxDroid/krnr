#!/usr/bin/env bash
set -euo pipefail

echo "Running gofmt -w ."
gofmt -w .

if command -v goimports >/dev/null 2>&1; then
  echo "Running goimports -w ."
  goimports -w .
else
  echo "goimports not found; you can install it with: 'go install golang.org/x/tools/cmd/goimports@latest'"
fi
