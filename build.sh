#!/usr/bin/env bash
set -euo pipefail

NAME="${1:-p2pchat}"

echo "Building for all platforms..."

build() {
  local os="$1" arch="$2" suffix="${3:-}"
  local binary="${NAME}-${os}-${arch}${suffix}"
  echo "  ${binary}"
  GOOS="$os" GOARCH="$arch" go build -o "build/${binary}" .
}

mkdir -p build

build linux   amd64
build linux   arm64
build darwin  amd64
build darwin  arm64
build windows amd64 .exe

echo ""
echo "Done! Binaries in ./build/"
ls -lh build/
