#!/usr/bin/env bash
set -euo pipefail

echo "Running golangci-lint..."
LOCAL_INSTALLED=true
if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint not found locally. Will attempt Docker fallback if available."
  LOCAL_INSTALLED=false
fi

# Run local linter if present
OUT=""
RC=0
if [ "$LOCAL_INSTALLED" = true ]; then
  set +e
  OUT=$(golangci-lint run ./... 2>&1)
  RC=$?
  set -e
  if [ $RC -eq 0 ]; then
    echo "No linter issues found (local)."
    exit 0
  fi
  echo "$OUT"
fi

# If local linter is missing or failed with export-data error, try Docker fallback
NEEDS_DOCKER=false
if [ $RC -ne 0 ]; then
  if echo "$OUT" | grep -q "unsupported version" || echo "$OUT" | grep -q "could not load export data" || [ "$LOCAL_INSTALLED" = false ]; then
    NEEDS_DOCKER=true
  fi
fi

if [ "$NEEDS_DOCKER" = true ]; then
  if command -v docker >/dev/null 2>&1; then
    echo "Attempting Docker-based golangci-lint (image: golangci/golangci-lint:v1.55.2)..."
    set +e
    docker run --rm -v "$(pwd)":/app -w /app golangci/golangci-lint:v1.55.2 golangci-lint run --verbose
    DOCK_RC=$?
    set -e
    if [ $DOCK_RC -eq 0 ]; then
      echo "Docker-based golangci-lint passed."
      exit 0
    else
      echo "Docker-based golangci-lint failed (exit $DOCK_RC)."
      exit $DOCK_RC
    fi
  else
    echo "Docker not available; cannot run docker fallback."
  fi
fi

# If we reach here, the local linter failed and Docker fallback wasn't possible or failed
if echo "$OUT" | grep -q "unsupported version" || echo "$OUT" | grep -q "could not load export data"; then
  echo "\ngolangci-lint encountered an export-data incompatible error. This usually means the linter binary was built with a different Go toolchain version than the one used to build the project. Try one of the following fixes:" 
  echo "  - Ensure the linter is installed with the same Go version (e.g., 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2' using your 'go' executable)."
  echo "  - Use the Docker fallback: 'docker run --rm -v \"$(pwd)\":/app -w /app golangci/golangci-lint:v1.55.2 golangci-lint run --verbose'"
  echo "  - Use the GitHub Action 'golangci/golangci-lint-action' in CI (this repo already does)."
  echo "  - Upgrade/downgrade your local Go toolchain to match the project's required Go version."
fi

exit ${RC:-1}
