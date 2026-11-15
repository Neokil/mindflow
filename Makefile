E2E_PROJECTS ?= chromium firefox chromium-http ipad-webkit tablet-chromium
E2E_SHARDS ?= 3

SQLC_VERSION := v1.31.1
GOLANGCI_VERSION := v2.15.0
GOVULNCHECK_VERSION := v1.9.0
ACTIONLINT_VERSION := v1.7.12
ACTIONLINT := $(CURDIR)/.cache/tools/actionlint/$(ACTIONLINT_VERSION)/actionlint
SQLC := $(CURDIR)/.cache/tools/sqlc/$(SQLC_VERSION)/sqlc
GOVULNCHECK := $(CURDIR)/.cache/tools/govulncheck/$(GOVULNCHECK_VERSION)/govulncheck
GOLANGCI := $(CURDIR)/.cache/tools/golangci-lint/$(GOLANGCI_VERSION)/golangci-lint

.PHONY: dev frontend frontend-build build lint lint-go lint-web test test-e2e verify-container tools generate check-generated audit upgrade

dev: native-codec
	MINDFLOW_DEV=1 MINDFLOW_ONENOTE_CODEC="$(CURDIR)/.cache/e2e/mindflow-onenote" go run ./cmd/mindflow

frontend:
	cd web && npm run dev

frontend-build:
	cd web && npm run build

build: frontend-build native-codec
	go build -trimpath -o mindflow ./cmd/mindflow
	cp .cache/e2e/mindflow-onenote ./mindflow-onenote

lint: lint-go lint-web lint-codec $(ACTIONLINT)
	$(ACTIONLINT)
	python3 -m py_compile scripts/*.py tests/container/server.py

lint-go: $(GOLANGCI) check-generated
	GOLANGCI="$(GOLANGCI)" bash scripts/check-go.sh

lint-web:
	cd web && npm run lint && npm run format:check

test: frontend-build native-codec
	MINDFLOW_ONENOTE_CODEC="$(CURDIR)/.cache/e2e/mindflow-onenote" go test -race ./cmd/... ./internal/...
	cd web && npm test
	python3 -m unittest discover -s scripts/tests

test-e2e: frontend-build native-codec
	mkdir -p .cache/e2e
	go build -o .cache/e2e/mindflow ./cmd/mindflow
	cd web && for project in $(E2E_PROJECTS); do shard=1; while [ "$$shard" -le "$(E2E_SHARDS)" ]; do npm run test:e2e -- --project="$$project" --shard="$$shard/$(E2E_SHARDS)" || exit $$?; shard=$$((shard + 1)); done; done

verify-container:
	docker build -t mindflow:verification .
	python3 scripts/verify-container.py --image mindflow:verification


tools: $(SQLC) $(GOLANGCI) $(ACTIONLINT)

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

$(GOVULNCHECK):
	mkdir -p "$(@D)"
	GOBIN="$(@D)" go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

audit: $(GOVULNCHECK)
	go mod verify
	$(GOVULNCHECK) ./cmd/... ./internal/...
	cd web && npm audit --audit-level=low
	python3 scripts/audit-native.py

upgrade:
	python3 scripts/upgrade.py $(UPGRADE_FLAGS)

$(ACTIONLINT):
	mkdir -p "$(@D)"
	GOBIN="$(@D)" go install github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)

# Manual README recording. The app and accounts exist only in a temporary fixture.
.PHONY: demo
demo: frontend-build $(CURDIR)/.cache/demo-venv/.ready
	mkdir -p .cache/demo
	go build -o .cache/demo/mindflow ./cmd/mindflow
	cd web && npx playwright install chromium
	node web/scripts/record-demo.mjs

$(CURDIR)/.cache/demo-venv/.ready: scripts/demo-requirements.txt
	python3 -m venv .cache/demo-venv
	.cache/demo-venv/bin/python -m pip install -r scripts/demo-requirements.txt
	touch "$@"

# Manual social announcement: actual UI capture, then a silent captioned MP4.
.PHONY: reel
reel: frontend-build $(CURDIR)/.cache/reel-venv/.ready
	mkdir -p .cache/reel
	go build -o .cache/reel/mindflow ./cmd/mindflow
	cd web && npx playwright install chromium
	node web/scripts/record-reel.mjs
	.cache/reel-venv/bin/python scripts/render-reel.py

$(CURDIR)/.cache/reel-venv/.ready: scripts/reel-requirements.txt
	python3 -m venv .cache/reel-venv
	.cache/reel-venv/bin/python -m pip install -r scripts/reel-requirements.txt
	touch "$@"

# Manual user-guide screenshots, using the same isolated fixture as the demo.
.PHONY: guide-images
guide-images: frontend-build
	mkdir -p .cache/demo
	go build -o .cache/demo/mindflow ./cmd/mindflow
	node web/scripts/record-demo.mjs --guide

.PHONY: native-codec
native-codec:
	bash scripts/build-onenote.sh

.PHONY: lint-codec
lint-codec:
	DOTNET_CLI_HOME="$(CURDIR)/.cache/dotnet-home" dotnet format whitespace internal/workspace/onenote/codec --folder --include Program.cs --verify-no-changes
