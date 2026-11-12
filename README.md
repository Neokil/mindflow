# mindflow

A minimalistic, self-hosted infinite canvas for images, shapes, text, and freehand drawing. Private workspaces, tabbed pages, and a quiet interface inspired by Apple applications.

## Run locally

Requires Go 1.26.11 and Node.js 22.12+ (the container build uses Node 26).

```sh
cd web
npm ci
npm run build
cd ..
MINDFLOW_DEV=1 go run ./cmd/mindflow
```

Open **http://localhost:8080**. On first startup, copy the one-time setup credential from the server log into the setup screen. Create the first administrator, enter the application URL, and configure SMTP with TLS/STARTTLS. `MINDFLOW_DEV=1` permits localhost HTTP; omit it behind your production HTTPS domain.

For frontend development, run `make dev` and `make frontend` in separate terminals, then open **http://localhost:5173**. Application data stays in `data/mindflow.sqlite`, including images; set `MINDFLOW_DATA` to change the directory.

## Features

- Invite-only email/password accounts, password recovery, and user administration.
- Private workspaces and reorderable page tabs; rename, duplicate, and delete both.
- Infinite pan/zoom, freehand lasso, shapes, formatted text, mouse drawing, and image placement.
- Resize/rotate, multi-select, grouping, stacking, copy/paste, session undo/redo, and optional grid snapping.
- Autosave, temporary disconnected editing, single-editor control with takeover, and recovery to a separate page.
- Original images and smaller display variants stored in SQLite; light/dark appearance.

Scroll to zoom around the cursor; Cmd/Ctrl + drag to pan. The keyboard button in the editor lists shortcuts. Browser support targets desktop Chrome and Firefox.

## Deploy

```sh
docker compose -f deploy/compose.yaml up --build -d
docker compose -f deploy/compose.yaml logs mindflow
```

The container serves HTTP on port 8080 and stores everything under `/data`. Route your existing Cloudflare Tunnel to that port and use your public HTTPS URL during setup. For TrueNAS, adapt [deploy/truenas.yaml](deploy/truenas.yaml) with your built image and dataset path; no public image has been published.

See [Running mindflow](docs/RUNNING.md) for TrueNAS installation, SMTP, snapshot/restore, and upgrade instructions.

## Verify

```sh
make lint
make test
make test-e2e
make verify-container
```

Go lint and sqlc tools install automatically into `.cache/tools` at pinned versions. After SQL changes, run `make generate` and commit the generated files.

The first browser-test run requires `cd web && npx playwright install chromium firefox`. Tests use temporary data and never contact a real email provider. Container verification requires Docker, Python 3 and OpenSSL; it runs an isolated HTTPS proxy and TLS SMTP fixture, then checks persistence and snapshot restore. GitHub Actions runs lint, tests, Chrome/Firefox acceptance, dependency auditing and container verification on pushes and pull requests.

Product/design documents: [requirements](docs/REQUIREMENTS.md), [architecture](docs/ARCHITECTURE.md), [canvas and saving](docs/CANVAS_AND_SYNC.md), [deployment design](docs/DEPLOYMENT.md), [implementation and validation](docs/IMPLEMENTATION.md), and [API contract](api/openapi.yaml).

Workspace sharing, export, and Linwood Butterfly import remain deferred.
