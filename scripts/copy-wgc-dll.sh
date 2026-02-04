#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DLL_PATH="${ROOT_DIR}/native/wgc_screenshot/build/Release/wgc_screenshot.dll"

if [[ ! -f "${DLL_PATH}" ]]; then
  echo "wgc_screenshot.dll not found: ${DLL_PATH}" >&2
  echo "Build it first:"
  echo "  cd native/wgc_screenshot && mkdir -p build && cd build"
  echo "  cmake .. -G \"Visual Studio 17 2022\" -A x64"
  echo "  cmake --build . --config Release"
  exit 1
fi

TARGET_DIR="${ROOT_DIR}/build/bin"
mkdir -p "${TARGET_DIR}"
cp -f "${DLL_PATH}" "${TARGET_DIR}/"

echo "Copied wgc_screenshot.dll to ${TARGET_DIR}"
