.PHONY: dev frontend build test test-e2e
dev:
	MINDFLOW_DEV=1 go run ./cmd/mindflow
frontend:
	cd web && npm run dev
build:
	cd web && npm ci && npm run build
	go build -o mindflow ./cmd/mindflow
test:
	go test ./...
	cd web && npm test && npm run build

test-e2e:
	cd web && npm run build
	mkdir -p .cache/e2e
	go build -o .cache/e2e/mindflow ./cmd/mindflow
	cd web && npm run test:e2e
