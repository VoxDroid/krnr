#!/usr/bin/env bash
set -euo pipefail

# Simple cross-compile helper
# Usage: ./scripts/build.sh linux/amd64 windows/amd64 darwin/amd64

for target in "$@"; do
  IFS='/' read -r GOOS GOARCH <<< "$target"
  echo "Building for $GOOS/$GOARCH"
  env GOOS=$GOOS GOARCH=$GOARCH go build -o "dist/krnr-$GOOS-$GOARCH" ./...
done
