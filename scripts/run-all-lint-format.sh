#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> golangci-lint fmt"
(cd "$repo_root" && GOCACHE=/tmp/go-build GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache golangci-lint fmt)

echo "==> golangci-lint run"
(cd "$repo_root" && GOCACHE=/tmp/go-build GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache golangci-lint run)

echo "==> frontend lint"
(cd "$repo_root/frontend" && bun run lint)

echo "==> frontend format"
(cd "$repo_root/frontend" && bun run format)
