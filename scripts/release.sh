#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Usage: $0 <version> [targets...]"
  echo "Example: $0 v0.1.0 linux/amd64 windows/amd64 darwin/amd64"
  exit 1
fi

VERSION="$1"
shift || true

# Default targets if none provided
if [ $# -eq 0 ]; then
  TARGETS=("linux/amd64" "linux/arm64" "windows/amd64" "darwin/amd64" "darwin/arm64")
else
  TARGETS=("$@")
fi

DISTDIR="dist"
rm -rf "$DISTDIR"
mkdir -p "$DISTDIR"

echo "Building release $VERSION"

for t in "${TARGETS[@]}"; do
  IFS='/' read -r GOOS GOARCH <<< "$t"
  BIN_NAME="krnr-${VERSION}-${GOOS}-${GOARCH}"
  EXT=""
  if [ "$GOOS" = "windows" ]; then
    EXT=".exe"
  fi
  OUT_PATH="$DISTDIR/${BIN_NAME}${EXT}"

  echo "Building $OUT_PATH"
  # Embed version into binary
  env GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "-s -w -X github.com/VoxDroid/krnr/internal/version.Version=${VERSION}" -o "$OUT_PATH" .

  # Package
  if [ "$GOOS" = "windows" ]; then
    ZIP_NAME="${BIN_NAME}.zip"
    (cd "$DISTDIR" && zip -r "$ZIP_NAME" "${BIN_NAME}${EXT}")
    rm -f "$OUT_PATH"
  else
    TAR_NAME="${BIN_NAME}.tar.gz"
    (cd "$DISTDIR" && tar czf "$TAR_NAME" "${BIN_NAME}")
    rm -f "$OUT_PATH"
  fi
done

# Generate checksums (include only existing archives)
pushd "$DISTDIR" > /dev/null
# collect existing artifacts (works correctly when one of the globs is missing)
files=( *.tar.gz *.zip )
existing=()
for f in "${files[@]}"; do
  if [ -e "$f" ]; then
    existing+=("$f")
  fi
done
if [ ${#existing[@]} -eq 0 ]; then
  echo "No release artifacts to checksum"
else
  shasum -a 256 "${existing[@]}" > "krnr-${VERSION}-SHA256SUMS"
fi
popd > /dev/null

echo "Release artifacts generated in $DISTDIR"
ls -la "$DISTDIR"

echo "Done."