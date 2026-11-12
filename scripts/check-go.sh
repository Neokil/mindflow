#!/usr/bin/env bash
set -euo pipefail
unformatted=$(gofmt -l cmd internal web/embed.go)
if [[ -n "$unformatted" ]]; then
  printf 'Run gofmt on:\n%s\n' "$unformatted"
  exit 1
fi
go vet ./cmd/... ./internal/...
