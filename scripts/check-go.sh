#!/usr/bin/env bash
set -euo pipefail
: "${GOLANGCI:?Run make lint-go to select the pinned tool version}"
"$GOLANGCI" config verify
"$GOLANGCI" run ./cmd/... ./internal/... ./web
format_diff=$("$GOLANGCI" fmt --diff ./cmd/... ./internal/... ./web)
if [[ -n "$format_diff" ]]; then
  printf '%s\n' "$format_diff"
  exit 1
fi
