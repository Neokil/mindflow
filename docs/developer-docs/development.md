# Development workflow

Use Go matching [go.mod](../../go.mod), Node.js 22.12+ with npm, and Python 3 for repository scripts. Tool/dependency versions are pinned in the manifests, Makefile, and container build rather than repeated here.

## Local setup

```sh
cd web && npm ci && cd ..
make frontend-build
make dev
```

Open `http://localhost:8080` and complete setup using the credential in the server log. `make dev` enables local HTTP. For frontend live reload, run `make frontend` in another terminal and open `http://localhost:5173`; Vite proxies `/api` and `/health` to the backend.

Data defaults to `data/mindflow.sqlite`; set `MINDFLOW_DATA` to use another directory. `make build` rebuilds the frontend and embeds it into a release binary. Rebuild Go after frontend changes when testing the bundled app. Docker is needed only for container verification/deployment, not normal local editing.

## Making a change

1. Check the [product definition](../product/overview.md) and [roadmap](../product/roadmap.md). Describe the observable behavior and relevant permission/data implications.
2. Put backend behavior in its owning domain and database code in that top-level domain's storage. Keep frontend components focused; reuse the per-page session protocol instead of adding parallel save/control logic.
3. Add tests for meaningful behavior and failure boundaries. Use small service/geometry tests where they suffice and browser coverage for interactions that depend on actual browser behavior.
4. Run the checks appropriate to the change, update the affected current guides/API contract, and commit coherent milestones separately.

## Database changes

Append an `Up`/`Down` Goose SQL file under `internal/<domain>/storage/migrations`; never edit an already-applied migration. Subdomains use their parent's schema and history. Add SQL queries to that parent's `storage/queries`, then run:

```sh
make generate
```

Commit migrations, SQL, sqlc bindings, and the generated migration catalog together. A new migration package exposes its embedded `Files()` and is discovered by `cmd/genmigrations`. `make lint` checks catalog/query freshness. Schema changes must preserve domain independence and existing data. Deployment rollback uses a matching image/snapshot; the app does not expose partial `down` maintenance commands.

## Verification

| Command | Coverage |
| --- | --- |
| `make lint` | Go linters/formatting, migration/sqlc freshness, ESLint/hooks, Prettier, workflow validation, and Python syntax. |
| `make test` | Frontend build, Go race tests with real temporary SQLite databases, frontend unit tests, and script tests. |
| `make audit` | Go module integrity, reachable known Go vulnerabilities, and npm advisories. |
| `make test-e2e` | Desktop Chrome/Firefox, insecure HTTP, and iPad WebKit/tablet Chromium interaction workflows. |
| `make verify-container` | Scratch runtime, migrations, HTTPS cookies/SSE, TLS mail, persistence, restore, and non-root/read-only operation. |

The Playwright `firefox` project tests desktop Firefox. The `ipad-webkit` project emulates an iPad in Playwright WebKit; it does not run the Firefox iPad app or reproduce that app’s browser-level restrictions. Fullscreen tests cover native desktop Chrome/Firefox and simulated permission/failure cases. Physical-device checks are required for browser-specific iPad behavior. Firefox for iOS currently restricts fullscreen for non-video content; track [Mozilla issue #33992](https://github.com/mozilla-mobile/firefox-ios/issues/33992).

Install browsers once with `cd web && npx playwright install chromium firefox webkit`. Browser fixtures bind to `127.0.0.1:8087` and use disposable accounts/data and a non-delivering SMTP target. Container checks require Docker, Python 3, and OpenSSL. No production token lookup endpoint exists; invitation tests read only their disposable email queue. Failure artifacts remain under ignored cache/test-result directories.

Each browser project runs two shards, each with a fresh disposable backend, to avoid consuming production authentication rate limits with unrelated test sign-ins. Set `E2E_SHARDS=1` for a small targeted run. Run a subset with `make test-e2e E2E_PROJECTS=ipad-webkit` (or a space-separated list). CI runs the projects as independent matrix jobs.

Tablet tests cover drag creation, pen strokes, concurrent fingers, two-finger navigation, cancellation, moving/resizing, text entry, image placement, properties, and comments. Chromium uses trusted CDP touch/pen input. Playwright WebKit has no multi-touch/Pencil injection API: those tests dispatch Pointer Events plus legacy touch events for Konva, with real browser taps on controls. These tests validate application behavior and WebKit rendering, but do not replace testing Safari and an actual Apple Pencil on an iPad. Check browser chrome/pull-to-refresh, software keyboard behavior, orientation changes, and palm contact on hardware before a tablet release.

`make tools` installs pinned tools under `.cache/tools`. Go lint enables all linters except the explicit exclusions in [.golangci.yml](../../.golangci.yml). Narrow suppressions need a specific reason; generated bindings should be regenerated rather than hand-edited. Architecture tests enforce domain storage and SQL ownership boundaries.

## CI and releases

[CI](../../.github/workflows/ci.yml) runs quality, browser, and container jobs on changes and weekly. Actions are pinned to commit SHAs, and failed browser cases upload artifacts. After successful checks, pushes to `main` publish `neokil/mindflow:edge` plus a commit tag; semantic version tags publish versioned images. Images target Linux amd64/arm64 and include provenance/SBOM attestations. The publishing job uses `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`; pull requests do not publish.

The [wiki workflow](wiki.md) publishes docs on `main`. [Dependency maintenance](dependencies.md) describes `make upgrade`. [Demo recording](demo.md) describes the manual README GIF workflow; it is not a CI job. Keep [user operations](../user-docs/operations.md) current when deployment or storage behavior changes.
