.PHONY: dev frontend build test
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
