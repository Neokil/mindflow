.PHONY: dev frontend frontend-build build lint lint-go lint-web test test-e2e verify-container

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

lint-go:
	bash scripts/check-go.sh

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
