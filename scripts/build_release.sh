#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

VERSION=$(grep -E '^[[:space:]]*Version[[:space:]]*=' pkg/registry/registry.go | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "failed to determine registry version" >&2
  exit 1
fi

DIST_DIR="$ROOT_DIR/dist/release-$VERSION"
STAGE_DIR="$ROOT_DIR/release/package/coreui-v$VERSION"
ZIP_PATH="$ROOT_DIR/release/coreui-v$VERSION.zip"
CONTEXT_PATH="$ROOT_DIR/release/CONTEXT_TEMPLATE.md"

rm -rf "$DIST_DIR" "$STAGE_DIR" "$ZIP_PATH"
mkdir -p "$DIST_DIR" "$STAGE_DIR"

go test ./...
go test ./tests -run TestSentinelGuardrails -count=1
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
  "linux amd64" \
  "darwin amd64" \
  "darwin arm64"
do
  set -- $target
  build_binary corec "$1" "$2"
  build_binary coredoc "$1" "$2"
done

go run ./cmd/corec context > "$CONTEXT_PATH"

cp COMPONENTS.md "$STAGE_DIR/"
cp release/WIRING.md "$STAGE_DIR/"
cp release/templates/SYSTEM_PROMPT.txt "$STAGE_DIR/"
cp "$CONTEXT_PATH" "$STAGE_DIR/"
cp "$DIST_DIR"/* "$STAGE_DIR/"

if command -v zip >/dev/null 2>&1; then
  (
    cd "$ROOT_DIR/release/package"
    zip -rq "$ZIP_PATH" "coreui-v$VERSION"
  )
elif command -v powershell.exe >/dev/null 2>&1; then
  STAGE_WIN=$STAGE_DIR
  ZIP_WIN=$ZIP_PATH
  if command -v cygpath >/dev/null 2>&1; then
    STAGE_WIN=$(cygpath -w "$STAGE_DIR")
    ZIP_WIN=$(cygpath -w "$ZIP_PATH")
  fi
  powershell.exe -NoProfile -Command "Compress-Archive -Path '$STAGE_WIN\\*' -DestinationPath '$ZIP_WIN' -Force"
else
  echo "zip or powershell.exe is required to package the release" >&2
  exit 1
fi

echo "release package written to $ZIP_PATH"
