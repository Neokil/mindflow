<p align="center">
  <img src="web/public/logo.png" alt="mindflow logo" width="100" />
</p>

<h1 align="center">mindflow</h1>

<p align="center">A little space for your thoughts.</p>

<p align="center">
  <a href="https://github.com/Neokil/mindflow/actions/workflows/ci.yml"><img src="https://github.com/Neokil/mindflow/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="docs/RUNNING.md">Documentation</a> · <a href="https://github.com/Neokil/mindflow/issues">Report an issue</a>
</p>

mindflow is a minimal, self-hosted infinite canvas for images, shapes, text, and freehand drawing. Organize ideas in workspaces with tabbed pages, invite people you trust, and discuss details directly on the canvas. Workspaces are private by default.

## Installation

Run mindflow with Docker Compose:

```yaml
services:
  mindflow:
    image: neokil/mindflow:edge
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - mindflow-data:/data
    read_only: true
    tmpfs:
      - /tmp:size=64m,mode=1777
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    restart: unless-stopped
    stop_grace_period: 30s
volumes:
  mindflow-data:
```

Save this as `compose.yaml`, then run:

```sh
docker compose up -d
docker compose logs mindflow
```

Route an HTTPS reverse proxy or Cloudflare Tunnel to the HTTP service on port **8080**. Open your public HTTPS URL, enter the one-time setup credential from the log, create the first administrator, and configure SMTP with TLS/STARTTLS for invitations and password recovery. If your tunnel runs in Docker, connect the containers through a private Docker network rather than the host's loopback binding.

All accounts, canvases, and images are stored in the `/data` volume. The runtime uses `scratch` and runs as UID/GID **10001**. `edge` follows the main branch; release builds use version tags.

For TrueNAS Apps, use [the installation guide](docs/RUNNING.md) and [Custom App YAML](deploy/truenas.yaml). The guide also covers SMTP, snapshots, restore, and updates. Read the [security notes](SECURITY.md) before exposing your instance.

## See it in action

![Creating a workspace, adding an image and elements, sharing access, and commenting in mindflow](docs/media/workflow.gif)

## Features

- **Room for ideas:** infinite pan and cursor-centered zoom, tabbed pages, a dot grid, and optional snapping.
- **Create freely:** drag to size images, shapes, and text; draw with the mouse. Crop, rotate, group, duplicate, and arrange elements.
- **Share deliberately:** invite-only accounts and private workspaces, with read-only, comment, or full access for selected users.
- **Discuss in context:** positioned comments, replies, avatars, and resolution controls.
- **Stay in control:** one writer per page, approval-based control requests, owner force control, and recovery of displaced unsaved work.
- **Keep your place:** workspace/page links, autosave, temporary disconnected editing, and a banner when a new build is available.
- **Make it yours:** light/dark appearance and a quiet interface designed for desktop Chrome and Firefox.

Scroll to zoom. Cmd/Ctrl + drag or hold the middle mouse button to pan. Hold Shift while creating an element for equal width and height. The editor's keyboard button lists more shortcuts.

Export and Linwood Butterfly import are planned for later.

## Development

Requires Go matching [go.mod](go.mod), Node.js **22.12+**, and npm. The container build pins its toolchain versions.

```sh
cd web && npm ci && cd ..
make frontend-build
make dev
```

Open **http://localhost:8080**. Development mode permits local HTTP; use production mode behind your HTTPS domain. For live frontend development, run `make frontend` in another terminal and open **http://localhost:5173**. Local data defaults to `data/mindflow.sqlite`; set `MINDFLOW_DATA` to change it.

```sh
make lint
make test
make audit
make test-e2e
make verify-container
```

Install browser binaries once with `cd web && npx playwright install chromium firefox`. Container checks require Docker, Python 3, and OpenSSL. CI runs quality, security, browser, and container checks on changes and weekly.

Run `make generate` after editing SQL or migration packages and commit the generated files. See [architecture](docs/ARCHITECTURE.md), [canvas and saving](docs/CANVAS_AND_SYNC.md), [requirements](docs/REQUIREMENTS.md), and the [API contract](api/openapi.yaml). `make upgrade` refreshes dependencies and pinned build references; see [upgrade instructions](docs/upgrading.md).

### Refresh the demo

```sh
make demo
```

This manually runs the [Playwright recording script](web/scripts/record-demo.mjs) against an isolated temporary instance, then writes `docs/media/workflow.gif`. It requires Python 3.11+; the command installs its pinned GIF encoder into `.cache/demo-venv`. Raw Playwright video stays under `.cache/demo/video/`. It never uses production accounts, data, or email delivery, and is not run automatically. See [recording details](docs/DEMO.md).
