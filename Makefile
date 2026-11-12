SQLC_VERSION := v1.31.1
GOLANGCI_VERSION := v2.15.0
SQLC := $(CURDIR)/.cache/tools/sqlc/$(SQLC_VERSION)/sqlc
GOLANGCI := $(CURDIR)/.cache/tools/golangci-lint/$(GOLANGCI_VERSION)/golangci-lint

.PHONY: dev frontend frontend-build build lint lint-go lint-web test test-e2e verify-container tools generate check-generated

dev:
	MINDFLOW_DEV=1 go run ./cmd/mindflow

frontend:
	cd web && npm run dev

frontend-build:
	cd web && npm run build

build: frontend-build
	go build -trimpath -o mindflow ./cmd/mindflow

lint: lint-go lint-web
	python3 -m py_compile scripts/verify-container.py tests/container/server.py

lint-go: $(GOLANGCI) check-generated
	GOLANGCI="$(GOLANGCI)" bash scripts/check-go.sh

lint-web:
	cd web && npm run lint && npm run format:check

test: frontend-build
	go test -race ./cmd/... ./internal/...
	cd web && npm test

test-e2e: frontend-build
	mkdir -p .cache/e2e
	go build -o .cache/e2e/mindflow ./cmd/mindflow
	cd web && npm run test:e2e

verify-container:
	docker build -t mindflow:verification .
	python3 scripts/verify-container.py --image mindflow:verification


tools: $(SQLC) $(GOLANGCI)

$(SQLC):
	mkdir -p "$(@D)"
	GOBIN="$(@D)" go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)

$(GOLANGCI):
	mkdir -p "$(@D)"
	GOBIN="$(@D)" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)

generate: $(SQLC)
	go run ./cmd/genmigrations
	$(SQLC) generate

check-generated: $(SQLC)
	go run ./cmd/genmigrations -check
	$(SQLC) diff
