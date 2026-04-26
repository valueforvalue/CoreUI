#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

VERSION=$(grep -E '^[[:space:]]*Version[[:space:]]*=' pkg/registry/registry.go | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "failed to determine registry version" >&2
  exit 1
fi

DIST_DIR="$ROOT_DIR/dist/$VERSION"
mkdir -p "$DIST_DIR"

go run ./cmd/coredoc

build_binary() {
  name=$1
  goos=$2
  goarch=$3

  suffix=""
  if [ "$goos" = "windows" ]; then
    suffix=".exe"
  fi

  output="$DIST_DIR/${name}_${VERSION}_${goos}_${goarch}${suffix}"
  echo "building $output"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -o "$output" "./cmd/$name"
}

for target in \
  "windows amd64" \
  "windows arm64" \
  "linux amd64" \
  "darwin arm64"
do
  set -- $target
  build_binary corec "$1" "$2"
  build_binary coredoc "$1" "$2"
done

echo "release artifacts written to $DIST_DIR"
