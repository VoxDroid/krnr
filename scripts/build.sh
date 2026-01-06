#!/usr/bin/env bash
set -euo pipefail

# Simple cross-compile helper
# Usage: ./scripts/build.sh linux/amd64 windows/amd64 darwin/amd64

for target in "$@"; do
  IFS='/' read -r GOOS GOARCH <<< "$target"
  echo "Building for $GOOS/$GOARCH"
  # read version from VERSION file or use dev placeholder
  VERSION=$(cat VERSION 2>/dev/null || echo "v0.0.0-dev")
  env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-s -w -X github.com/VoxDroid/krnr/internal/version.Version=${VERSION}" -o "dist/krnr-$GOOS-$GOARCH" .
done
