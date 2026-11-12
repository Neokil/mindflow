# mindflow — Deployment and operations

Status: deployment design draft, 12 November 2026. Implementation and manifests are now available; use [RUNNING.md](RUNNING.md) for current commands and installation steps. This document preserves the deployment design.

Related documents: [Architecture](ARCHITECTURE.md), [Canvas and synchronization](CANVAS_AND_SYNC.md), [Requirements](REQUIREMENTS.md).

## 1. Deployment model

One mindflow container runs the Go backend and serves the bundled frontend. SQLite stores all durable application state, including original images, variants, SMTP configuration, accounts, and jobs.

The owner installs Compose-compatible YAML through TrueNAS: Apps → Discover Apps → three-dot menu → Custom App → Install via YAML. The user's TrueNAS version remains unspecified; validate the exact manifest on that installed version before release.

The existing Cloudflare Tunnel routes the public HTTPS hostname to the application's HTTP listener. mindflow does not manage Cloudflare, certificates, a tunnel container, or a reverse proxy. Cloudflare-provided identity is not a replacement for the application's invite-only authentication.

## 2. Container contract

Proposed configuration contract:

| Item | Default / behavior |
| --- | --- |
| HTTP listener | `0.0.0.0:8080` inside the container; configurable. |
| Persistent data directory | `/data`. |
| Database | `/data/mindflow.sqlite`, with SQLite sidecar files in the same directory. |
| Process | One Go process, non-root user, one replica. |
| Public URL | Configured in setup; required HTTPS URL in production. |
| SMTP | Configured in setup and editable in administrator settings. |
| Restart policy | Restart unless explicitly stopped; failed migrations must remain visible. |
| Shutdown grace | Proposed 30 seconds, validated against workload. |
| Health endpoints | `/health/live` and `/health/ready`, no private information. |

Use a host-path dataset mount for `/data`, with ownership matching the container UID/GID. Use a local dataset filesystem with working SQLite locking; do not put the database on an SMB/NFS share. Do not run multiple containers against the same file.

The release build should support Linux amd64 initially; add arm64 only after image processing and SQLite-driver compatibility are tested. Use a multi-stage build: Node tooling builds frontend assets, Go builds the application, and the runtime includes only the binary and required runtime files such as CA certificates and timezone data. The runtime needs no Node installation.

Minimize environment settings to listener, data path, logging, trusted proxy configuration, and optional explicit development mode. Durable administrator-configured settings live in SQLite. No separate image volume, database container, SMTP container, or Redis is required.

## 3. Public HTTP origin behind HTTPS

Use the configured public HTTPS URL for email links, origin validation, and secure cookie policy. The internal HTTP hop is acceptable only within the operator's trusted host/network path to the tunnel connector; avoid exposing the origin port directly to the public internet.

Configure only known tunnel/reverse-proxy peers as trusted forwarding sources. Do not trust client-supplied proxy headers from arbitrary connections. The server must send production secure cookies even though its immediate connection is HTTP.

Exclude `/api/*`, events, and all protected images from Cloudflare caching. Do not inject cache rules that publish authenticated responses. Verify that long-lived SSE streams flush promptly and survive normal tunnel operation; reconnect and polling fallback are required regardless.

Body limits should account for a 20 MiB image plus multipart overhead, while rejecting excess bytes early. JSON document limits are separate. Use explicit upload/read-header/idle timeouts and SSE-specific streaming behavior rather than applying a short global response timeout to event streams.

Development can explicitly permit localhost HTTP and a development cookie configuration. It must not silently weaken production behavior based only on an untrusted Host or forwarding header.

## 4. Setup and email

On an uninitialized database, the operator receives a one-time bootstrap credential through a local server-operator mechanism. Opening the setup screen requires that credential to complete initialization. It must not be exposed through unauthenticated status endpoints or access logs. Delivery details will be implemented and documented before release.

Setup collects:

- First administrator email and password.
- Public application URL, for example `https://mindflow.example.com`.
- SMTP host, port, transport security mode, optional username/password, sender address, and sender name.

Support implicit TLS or required STARTTLS with certificate verification; do not silently downgrade to plaintext. Provide a test-email action. A failed test is visible and retryable without creating multiple administrators. Persist initialization atomically and disable first-run completion afterward.

Administrators can later edit SMTP/public URL settings and send test emails. Blank SMTP-password fields mean unchanged unless an explicit clear action is provided. Never return a stored SMTP password to the browser.

Use a database-backed email queue with bounded retry/backoff and visible failure status in user management. Invitation/reset state and the corresponding email job must commit together. Queue payloads that need raw one-time tokens must be treated as secrets, accessible only to the application and purged after delivery, terminal failure, expiration, or invalidation. Never log them. Delivery retries may send duplicates; repeated valid link submissions must still consume the token only once. Do not send links after they expire or have been superseded.

All SMTP secrets necessarily remain accessible to the server process and database operator. Protect the dataset and snapshots accordingly; application administrator status alone does not permit database access.

## 5. SQLite behavior and disk usage

Proposed connection configuration:

- WAL journal mode.
- Foreign-key enforcement on every connection.
- `synchronous=FULL` for durable save acknowledgment.
- A bounded busy timeout, proposed five seconds, and short write transactions.
- One serialized writer connection; a small read pool, initially at most four connections.
- Controlled checkpointing and monitoring of database/WAL growth.

SQLite settings must be validated against the chosen driver. Foreign-key enforcement is an explicit connection requirement, not an assumed default. [SQLite pragma documentation](https://www.sqlite.org/pragma.html)

Perform decoding, resizing, email network calls, and expensive validation outside database write transactions. Commit derived image results only after rechecking that the asset/user still exists. Original uploads remain durable even when generation fails; surface errors and retry safely.

Blob storage simplifies complete snapshots but increases database size, WAL traffic, and working memory. Images and variants must be streamed or bounded where practical; do not read all media into memory when listing/opening pages. Monitor disk free space and fail saves explicitly on exhaustion. Never mark an edit saved after a failed commit.

Deleted content is removed from live application records and no longer retrievable through the app. SQLite can reuse free pages without reducing file size. Controlled compaction may be documented later; do not run full vacuum automatically on every deletion. Deletion is not a promise of forensic erasure from old database pages, WAL, or retained snapshots.

## 6. Backups and restore

TrueNAS handles snapshot schedules, retention, and any replication. mindflow does not add backup scheduling or an application backup/export command in the first release.

The snapshot unit is the complete persistent data directory on one dataset, including the database and its WAL/sidecar files. A live filesystem snapshot must capture these together at one point in time. Copying only `mindflow.sqlite` while writes continue is not a valid backup procedure; WAL can contain committed data not yet checkpointed into the main file. [SQLite WAL documentation](https://sqlite.org/wal.html)

The simpler fallback is to stop the app cleanly before taking a complete dataset snapshot. Validate live snapshot restore on the actual TrueNAS storage configuration before relying on it. Backups include original media, generated variants, users, settings, and queued work; there is no separate media tree to synchronize.

Proposed restore procedure:

1. Stop mindflow and prevent a second instance from starting.
2. Restore the complete data directory from one snapshot, preserving file ownership and permissions. Do not combine database/WAL files from different points in time.
3. Start the application version compatible with that snapshot's schema. Check migration/readiness status and SQLite integrity and foreign-key checks.
4. Invalidate restored sessions, invitation/reset tokens, pending token-bearing emails, and editor leases before reopening public access. Implementation must provide an operator-only maintenance action for this; a restored old session/token must not regain access merely because its prior revocation was rolled back.
5. Check login, a representative workspace, original/variant images, SMTP configuration, and new saving. Keep the pre-restore snapshot available until validation finishes.

Restore may resurrect users/content deleted after the snapshot. Operators must account for those changes before reopening access. TrueNAS retention determines how long deleted content remains in backups. These are consequences of snapshot restore, not additional application history features.

## 7. Upgrades and shutdown

Before upgrades, take a dataset snapshot. Stop the old instance, start the pinned new image, apply ordered schema migrations, and only report ready after migration success. A failed migration must not allow requests against a partially migrated schema. Avoid destructive migrations without an explicit forward/restore plan.

Use transactional migrations where supported, with recorded schema versions. Older binaries must refuse incompatible newer schemas. Downgrades use a matching pre-upgrade snapshot and image, not an unsupported reverse migration.

On shutdown, stop admitting mutations, finish or reject in-flight transactions coherently, cancel background work safely, close event streams, and close SQLite connections. Persisted jobs resume after restart; lease epochs are invalidated. Browsers reconnect and verify revisions/control before sending edits.

## 8. Health, logs, and resource targets

Liveness indicates the process can respond. Readiness indicates migrations completed and the database is accessible; an uninitialized app can be ready to serve setup. SMTP failure is visible but should not make existing users' canvas editor unavailable.

Emit structured logs to stdout/stderr with request IDs, status codes, timing, migration results, job outcomes, and storage failures. Exclude passwords, cookies, CSRF values, invitation/reset tokens, SMTP secrets, image bytes, document bodies, and text content. Avoid logging token-containing URLs or query strings. Distinguish expected authorization denials from server faults.

Proposed starting allocation is two CPU cores and 1 GiB memory; this is a measurement starting point, not a guaranteed minimum. Bound image processing to one job, apply decode limits, and measure large-image memory use alongside the performance fixture. Runtime needs may be raised based on those results.

## 9. Delivery and verification

Implementation should add reproducible local-development instructions, a pinned multi-stage container build, Compose-compatible TrueNAS YAML, and an operator maintenance action for restore-time credential invalidation. Validate development proxying so frontend API requests remain consistent with production origin rules.

Required release checks:

- Go unit/service tests and real-SQLite integration tests.
- Frontend typecheck/build and geometry/document tests.
- Browser acceptance tests in Chrome and Firefox.
- Container startup, persistence across restart, and graceful shutdown.
- Fresh setup and upgrade from a prior fixture database.
- HTTPS public URL with HTTP origin, invitation/reset email flow, protected image caching, and SSE through the actual tunnel.
- TrueNAS dataset snapshot/restore drill, including media and credential invalidation.
- Disk-full handling and image worker limits.

## 10. Remaining implementation validation

The interview does not need to choose individual library versions. Before implementation is considered ready, record:

- Pinned Go/Node/frontend versions, SQLite driver, SMTP library, image codecs, and license review.
- Renderer prototype results and final Konva decision.
- TrueNAS version, host dataset path/UID/GID, origin port, public hostname, and trusted proxy peers.
- Measured performance/resource limits and any adjusted defaults.
- Final OpenAPI contract, SQL migrations, and restore maintenance command.

These are validation and implementation tasks, not permission to expand product scope.
