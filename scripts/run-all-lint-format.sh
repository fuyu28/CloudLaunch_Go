#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> golangci-lint fmt"
(cd "$repo_root" && GOCACHE=/tmp/go-build GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache golangci-lint fmt)

echo "==> golangci-lint run"
(cd "$repo_root" && GOCACHE=/tmp/go-build GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache golangci-lint run)

echo "==> native format"
if command -v clang-format >/dev/null 2>&1; then
  (cd "$repo_root" && clang-format -i native/wgc_screenshot/*.cpp)
  (cd "$repo_root" && clang-format -i native/dxgi_screenshot/*.cpp)
else
  echo "clang-format not found; skipping"
fi

echo "==> native lint"
if command -v clang-tidy >/dev/null 2>&1; then
  if [[ -f "$repo_root/native/wgc_screenshot/build/compile_commands.json" ]]; then
    (cd "$repo_root" && clang-tidy -p native/wgc_screenshot/build native/wgc_screenshot/*.cpp)
  else
    echo "compile_commands.json not found; skipping clang-tidy"
  fi
  if [[ -f "$repo_root/native/dxgi_screenshot/build/compile_commands.json" ]]; then
    (cd "$repo_root" && clang-tidy -p native/dxgi_screenshot/build native/dxgi_screenshot/*.cpp)
  else
    echo "compile_commands.json not found; skipping clang-tidy"
  fi
else
  echo "clang-tidy not found; skipping"
fi

echo "==> frontend lint"
(cd "$repo_root/frontend" && bun run lint)

echo "==> frontend format"
(cd "$repo_root/frontend" && bun run format)
