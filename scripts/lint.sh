#!/usr/bin/env bash
set -euo pipefail

echo "Running golangci-lint..."
if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint not found; install it locally (e.g. 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest') or use the GitHub Action 'golangci/golangci-lint-action'"
  exit 1
fi

set +e
OUT=$(golangci-lint run ./... 2>&1)
RC=$?
set -e
if [ $RC -ne 0 ]; then
  echo "$OUT"
  # Detect common export data / unsupported version errors and give guidance
  if echo "$OUT" | grep -q "unsupported version" || echo "$OUT" | grep -q "could not load export data"; then
    echo "\ngolangci-lint encountered an export-data incompatible error. This usually means the linter binary was built with a different Go toolchain version than the one used to build the project. Try one of the following fixes:" 
    echo "  - Ensure the linter is installed with the same Go version (e.g., 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2' using your 'go' executable)."
    echo "  - Use the GitHub Action 'golangci/golangci-lint-action' in CI (this repo already does)."
    echo "  - Upgrade/downgrade your local Go toolchain to match the project's required Go version."
  fi
  exit $RC
fi

echo "No linter issues found."
