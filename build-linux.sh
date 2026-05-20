#!/usr/bin/env bash
# Build script para tlog-pipeline en Linux.
# No requiere goversioninfo ni resource.syso (son específicos de Windows).
#
# Uso:
#   ./build.sh                        # versión por defecto, output: tlog-gen
#   ./build.sh 2.1.0                  # versión explícita
#   ./build.sh 2.1.0 tlog-gen-linux   # versión + nombre de salida

set -euo pipefail

VERSION="${1:-2.1.0}"
OUTPUT="${2:-tlog-gen}"

ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "Compilando $OUTPUT v$VERSION..."
go build \
    -ldflags "-s -w -X main.Version=${VERSION}" \
    -o "${ROOT}/${OUTPUT}" \
    ./cmd/pipeline

echo "OK: ${OUTPUT} v${VERSION}"
