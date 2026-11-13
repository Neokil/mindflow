# Running mindflow

mindflow runs as one Go application with an embedded React frontend and a SQLite database. Original images, variants, accounts, settings, and queued jobs are all in the database. Run one instance per database.

## Local development

Use Go 1.26.11 and a Node version accepted by Vite (22.12+; tested with 26.13.1). Install packages with `cd web && npm ci`. Start the backend with `make dev` and the frontend with `make frontend`. Open `http://localhost:5173`; Vite proxies API requests to port 8080.

The default backend data directory is `./data`. Release binaries require `make build` first; the frontend output is embedded at Go compilation time. Rebuild the Go binary after rebuilding the frontend when testing the bundled application.

Configuration:

| Setting | Default | Purpose |
| --- | --- | --- |
| `MINDFLOW_ADDR` | `:8080` | HTTP bind address. |
| `MINDFLOW_DATA` | `data` (container: `/data`) | Persistent database directory. |
| `MINDFLOW_DEV` | unset | Set to `1` only for local HTTP development. |

The service does not rely on forwarding headers for authentication, URL generation, or client-IP limits. Public URL comes from setup. Rate limiting currently uses the direct peer address, so users behind a tunnel share the limit. Do not add untrusted forwarded-IP support without a trusted-proxy policy.

## Container installation

From a checkout, run:

```sh
docker compose -f deploy/compose.yaml up --build -d
docker compose -f deploy/compose.yaml logs mindflow
```

The named volume `mindflow-data` stores `/data`. The container uses UID/GID `10001:10001`, a read-only root filesystem, and temporary writable `/tmp`. Its built-in health check uses `mindflow healthcheck`.

Production requires HTTPS at the browser. The application itself serves ordinary HTTP on the origin port. Point your existing Cloudflare Tunnel at the container/host's reachable HTTP address; `localhost` refers to the connector's own network namespace, so use a host or service address appropriate to your setup. Keep the origin port inside the trusted network and bypass Cloudflare caching for `/api/*`, including protected image responses and events. Verify SSE streaming; polling provides a fallback.

For local testing of a container over plain HTTP, explicitly set `MINDFLOW_DEV=1`. Production secure cookies intentionally do not work through a plain HTTP public URL.

### TrueNAS Custom App

1. Use `neokil/mindflow:<version>` after a successful publishing run, or build and push it yourself:

   ```sh
   docker build -t neokil/mindflow:0.1.0 .
   docker push neokil/mindflow:0.1.0
   ```

2. Create a local dataset for application data, for example `/mnt/POOL/apps/mindflow`. Grant UID/GID `10001:10001` read/write and directory traversal permissions. Do not put the database on an SMB/NFS mount.
3. Edit [deploy/truenas.yaml](../deploy/truenas.yaml), selecting the image tag and replacing `POOL`, the dataset path, and the host port as needed.
4. In Apps → Discover Apps → three-dot menu → Custom App → Install via YAML, paste the edited YAML.
5. Read the application startup log for its one-time setup credential. Open your configured public hostname and complete setup.
6. Verify login, saving across restart, SMTP test delivery, and snapshot restore on your installed TrueNAS version.

The publishing job targets `neokil/mindflow`; availability of an image requires a successful GitHub Actions publishing run. The host's Docker daemon must be running to build or test the container. An amd64 server needs an amd64 image: build on that architecture or use Docker Buildx with an explicit platform. The pure-Go application can be cross-compiled, but image builds must still be validated on the target.

## First-run setup and accounts

The server prints a random setup credential only while the database is uninitialized. It changes on restart until setup completes. Treat access to startup logs as operator access. Public setup status never reveals it.

Enter the credential, first administrator email/password, public application URL, and SMTP settings. Production URL must use HTTPS and have no extra path/query/fragment. You can send a test email before completing setup. The completed setup transaction prevents further first-administrator creation.

SMTP supports required STARTTLS or implicit TLS with certificate verification. Common ports are 587 (STARTTLS) and 465 (TLS), but use your provider's settings. Enter a permitted sender address. The app makes no changes to DNS or your email provider.

Administrators can invite users, resend invitations, trigger resets, deactivate/reactivate users, delete users, and assign roles. At least one active administrator must remain. Administration grants no workspace access; another owner can explicitly share a workspace with an administrator just like any user.

Invitations last seven days; reset links last one hour. Resending supersedes older links. Login sessions last 30 days unless revoked. Passwords must contain 15–128 characters. Email sending is queued in SQLite and retries failures; user management shows delivery problems. Check SMTP settings and resend after fixing a terminal failure. Stored SMTP passwords are not returned to the browser; a blank password on settings update retains the existing secret.

## Editing, saving, and recovery

Each page has one server-enforced editor. Other sessions show saved changes read-only and can explicitly take control. Page tabs preserve in-session undo and pending changes. Autosave is debounced by 750 ms; continuous edits are saved at most five seconds apart, excluding an unfinished pointer gesture and network/upload delays.

On connection loss, loaded content remains editable in memory. Reconnecting checks control and revision before saving. If another editor took over, the old tab keeps a recovery copy and offers saving it as an independent page. Do not close that tab until recovery is saved. Browser-native unsaved-change warnings are best-effort; forced termination/crashes do not preserve drafts.

Original PNG/JPEG/WebP uploads are retained, up to 20 MiB and 40 megapixels (maximum 16,384 pixels per side). Variants are generated asynchronously for smaller render sizes. Unsupported animated images are rejected. Files are never served publicly. Removing an image retains its asset until the retaining page is deleted so undo/recovery remain possible. Duplicates can share immutable images within the same account; deleting an original does not break a duplicate.

## Backup and restore

TrueNAS handles scheduling, retention, and replication. Snapshot the complete `/data` dataset atomically, including `mindflow.sqlite` and its WAL/sidecar files. Do not copy only the live `.sqlite` file. If live snapshot consistency has not been verified on your storage configuration, stop the app before taking the snapshot.

Restore procedure:

1. Stop the application and take a safety snapshot of its current dataset.
2. Restore the entire dataset from one snapshot, preserving ownership and permissions. Do not mix sidecar/database files from different snapshots.
3. Before reopening access, use the matching application image to check data and revoke restored credentials. With the local Compose deployment stopped:

   ```sh
   docker compose -f deploy/compose.yaml run --rm mindflow check-data
   docker compose -f deploy/compose.yaml run --rm mindflow revoke-credentials
   docker compose -f deploy/compose.yaml up -d
   ```

   On TrueNAS, perform the equivalent one-off commands against the restored dataset with the normal app stopped; do not start a second app against the same database. For a local binary, use `MINDFLOW_DATA=/path/to/data ./mindflow check-data` and `./mindflow revoke-credentials` with the same data setting.
4. Sign in again and verify workspace contents, images, new saves, SMTP, and user status.

`revoke-credentials` removes sessions, invite/reset links, queued token emails, and editor control. Passwords, users, workspaces, and media remain. Resend needed invitations afterward. Snapshot restore can resurrect accounts/content deleted after the snapshot; review those changes before exposing the app again.

Deleting live records does not immediately shrink SQLite files or erase retained snapshots. Infrastructure dataset permissions, encryption, retention, and access remain the operator's responsibility.

## Upgrades and diagnostics

Take a snapshot, stop the old container, deploy a pinned new image, and check readiness after migrations. Migrations are transactional and the app refuses newer unsupported database schemas. Rollback uses the old image plus its matching pre-upgrade snapshot. Do not assume an old binary can read an upgraded database.

`/health/live` reports process health; `/health/ready` verifies database access. An uninitialized instance can be ready to serve setup. SMTP failures do not take the canvas offline. Logs include server errors without document bodies, passwords, cookies, or email link tokens (apart from the intentionally printed first-run setup credential).

Image decoding and processing are serialized to bound memory; the database uses WAL, full synchronization, foreign keys, and one serialized connection. Start with two CPUs and 1 GiB memory, then measure your own image workload. This is a starting allocation rather than a guaranteed minimum. Keep enough free disk for originals, variants, WAL growth, and snapshots.

## Domain migrations

All schema changes run through Goose, with a separate version table for each domain. The application applies pending migrations before normal startup. To run them explicitly, stop the application first, then use the same data directory or volume:

```sh
MINDFLOW_DATA=./data go run ./cmd/mindflow migrate
# Or, after stopping the Compose service:
docker compose -f deploy/compose.yaml run --rm mindflow migrate
```

The command discovers the embedded domain catalog, applies pending migrations and exits. The four top-level schemas have no migration dependencies on one another. It starts no HTTP server and requires no account or SMTP setup. Repeated runs are safe. Keep one application/migration process against the SQLite file at a time.

History is independent: `goose_account_version`, `goose_workspace_version`, `goose_email_version` and `goose_installation_version`. Subdomains share their parent's schema and history. Initial development scripts were replaced during domain consolidation; databases from the previous layout are not supported. Existing consolidated databases upgrade in place; only the earlier pre-consolidation layout needs a fresh development database. The app never deletes an existing database automatically.

For a new migration, append an `Up`/`Down` Goose SQL file under `internal/<domain>/storage/migrations`; do not edit already applied files. Each migration package exposes its embedded `Files()`. When adding a domain migration package, run `make generate` (or `go generate ./internal/mindflow/storage`) to discover and link it automatically. Then run `make generate` for sqlc bindings and commit the SQL and generated files together. The binary embeds SQL, so release containers need no migration-directory mount or standalone Goose executable.

Each migration is atomic, but an upgrade across domains is not one transaction. If a migration fails, its own changes/history roll back; earlier successful domains stay applied and a retry resumes remaining work. Use the matching image and pre-upgrade snapshot for release rollback. The application does not expose `down` commands that could partially roll back collaborating domains.

## Tests

```sh
make lint
make test
cd web && npx playwright install chromium firefox
cd .. && make test-e2e
```

Browser tests run a temporary backend on `127.0.0.1:8087` with a temporary SQLite database and a local, non-delivering SMTP target. Test data is removed afterward. The additional `chromium-http` project maps `mindflow.test` to loopback inside Chromium to test an actual insecure HTTP origin without exposing a server to the LAN. It verifies `randomUUID` is unavailable and covers workspace opening, editing, saving and reopening. The regular Chrome/Firefox cases exercise setup/login, canvas composition, lasso/grouping, images/transforms/variants, persistence, two-session takeover with disconnected recovery, and a large-page fixture. Backend tests cover access isolation, token lifecycle, revocation/deletion, and save fencing/idempotency. Unit tests cover geometry and save-state edge cases.

`make verify-container` builds the production image and tests it with disposable Docker volumes, an HTTPS proxy and a capturing TLS SMTP server. Requires Docker, Python 3 and OpenSSL. It first runs the standalone `migrate` command twice, then checks secure cookies, SSE delivery, invitations, implicit TLS/STARTTLS, owner isolation, image originals/variants, save retries and takeover, restart persistence, snapshot restore and credential revocation. The application runs without root, with a read-only root filesystem. Fixture email stays inside the test environment; the script removes its own containers, network and volumes afterward.

`make tools` installs pinned sqlc 1.31.1 and golangci-lint 2.15.0 binaries under `.cache/tools/`. Goose 3.30.0 is a normal Go dependency. `make lint` validates linter configuration, checks formatting, checks the migration catalog and uses `sqlc diff` to reject stale query bindings. Each top-level domain's database code lives under `internal/<domain>/storage`; its subdomains use that same storage, including repository code, queries, generated files and migrations. `sqlc` reads the Goose migration directories as its schema source. Edit SQL rather than generated files.

`.golangci.yml` uses `default: all` and disables exactly `funlen`, `lll`, `depguard`, `exhaustruct`, `exhaustruct_v5`, `dupl`, `cyclop`, `maintidx`, `canonicalheader`, `wsl`, `wsl_v5` and `gomodguard`. Internal typed errors can pass through package boundaries; external failures receive operation context. JSON naming preserves the existing `publicURL` spelling. Narrow, explained gosec suppressions cover the fixed-loopback health check, bounded multipart parsing, deliberate development cookies and disposable test credentials. All other enabled checks apply to production and test code, apart from golangci-lint's standard generated-code handling.

GitHub Actions runs these checks on pushes and pull requests in three jobs: quality (pinned tools, migration catalog and SQL generation freshness, golangci-lint/formatting/race tests, ESLint/React hooks, Prettier, TypeScript/build, frontend unit tests and dependency audit), Chrome/Firefox acceptance, and production container verification. Actions are pinned to commit hashes. Browser failures upload test artifacts. Hosted runs start once these commits are pushed to GitHub.

The frontend package override pins `minimatch` to 11.0.8 to resolve development-tool dependency advisories. Keep the override under review when updating ESLint dependencies; `npm audit` is checked in CI.


## Docker Hub publishing

After quality, browser and container verification succeed, pushes to `main` publish `neokil/mindflow:edge` and a `sha-<short-commit>` tag. Pushing a semantic version tag such as `v0.1.0` publishes `neokil/mindflow:0.1.0`; stable versions also receive `latest`. Prerelease tags do not replace `latest`. Pull requests and manual CI runs do not publish images.

The image job uses the repository secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` to authenticate to Docker Hub. It builds Linux amd64 and arm64 images with commit-pinned official Docker actions, following the [Docker GitHub Actions workflow](https://docs.docker.com/build/ci/github-actions/). The image name is fixed to `neokil/mindflow`, independently of the login username. No publishing credentials are needed by the test jobs. Use a version tag or digest for stable deployments; the TrueNAS example defaults to `edge` for initial development.

## Element editing

In Select mode, hovering over movable elements shows a grab cursor and dragging shows a grabbing cursor. Text properties support whole-number font sizes (1–2,000) and horizontal/vertical alignment.

To crop, select one image, open Selection properties and choose **Crop image**. Draw or adjust the rectangle or trim percentages, then Apply. Cancel discards the draft. Reset crop followed by Apply restores the full original. Crops support undo/redo and retain the original file.


## Sharing and comments

Click **Private** beside the workspace name to share with an existing active user's email. Choose Read-only, Comment or Full access. The badge becomes **Shared**; only the owner manages its membership menu. Members find the workspace in their workspace list. Sharing sends no email invitations and creates no public link; account invitations remain administrator-managed. Removing the last member makes the workspace private again.

Use **Add comment** in the toolbox and click a canvas point to write a comment. Hover/focus a pin to read and reply. The thread creator or workspace owner can Resolve/Reopen/Delete the thread. Resolved pins can be shown using **Show resolved comments**. Commenting remains available to comment/full roles even while another user owns the canvas edit lease.

This update appends workspace migration 2 and upgrades existing databases automatically. Existing workspaces remain private. Shared image uploads remain with the workspace owner's content if the uploader's account is deleted. Workspace/page copies start without memberships or comment threads.

The multi-user browser tests create disposable invited accounts and read invitation tokens from the temporary fixture's queued email in SQLite using Node's built-in SQLite API. They exercise the actual invite acceptance UI; no token lookup route is exposed by the application and no real email is sent.


## Security and dependency maintenance

The production runtime uses `scratch`, a static Go binary, CA certificates and UID/GID 10001. There is no shell; invoke maintenance commands through the image entrypoint, or use `/usr/local/bin/mindflow` with `docker exec`. Keep the HTTP origin private behind your HTTPS tunnel and leave production mode enabled. See [SECURITY.md](../SECURITY.md) for boundaries and scanner limitations.

`make audit` checks module integrity and known Go/npm vulnerabilities. `make upgrade` resolves stable dependencies, pins Actions to commit SHAs and Docker references to digests, regenerates code, and runs the quality gate. See [upgrade options and review steps](upgrading.md). Avatar and control-request migrations are applied automatically during startup; snapshot the complete data directory before upgrading.
