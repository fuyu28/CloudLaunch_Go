#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_PATH="${ROOT_DIR}/native/dxgi_screenshot/build/Release/dxgi_screenshot.exe"

if [[ ! -f "${BIN_PATH}" ]]; then
  echo "dxgi_screenshot.exe not found: ${BIN_PATH}" >&2
  echo "Build it first:"
  echo "  cd native/dxgi_screenshot && mkdir -p build && cd build"
  echo "  cmake .. -G \"Visual Studio 17 2022\" -A x64"
  echo "  cmake --build . --config Release"
  exit 1
fi

TARGET_DIR="${ROOT_DIR}/build/bin"
mkdir -p "${TARGET_DIR}"
cp -f "${BIN_PATH}" "${TARGET_DIR}/"

echo "Copied dxgi_screenshot.exe to ${TARGET_DIR}"
