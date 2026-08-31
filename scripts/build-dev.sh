#!/usr/bin/env bash
# Build a local development binary as logyDEV (does not overwrite release logy).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${LOGY_DEV_VERSION:-0.0.0-dev}"
OUT="$ROOT/logyDEV"
LDFLAGS="-X logy/internal/version.Version=${VERSION}"

echo "Building $OUT (version $VERSION)..."
go build -ldflags "$LDFLAGS" -o "$OUT" ./cmd/logy
"$OUT" version
echo
echo "Run with: ./logyDEV <command>"
