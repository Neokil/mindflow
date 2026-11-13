# Operations

Run one application or migration process against a database at a time. Persistent data is in the configured data directory, including the SQLite database, its WAL/sidecar files, and all images. See [installation](installation.md) for paths and permissions.

## Backups

TrueNAS or your host handles snapshot scheduling, retention, and replication. Snapshot the complete `/data` dataset atomically; do not copy only the live `mindflow.sqlite` file. If live snapshot consistency is not verified for your storage configuration, stop the app first.

Protect backups as private data. They contain accounts, workspaces, images, SMTP credentials, and possibly pending credential-link emails. Deleting a live record does not erase retained snapshots or immediately shrink the database file.

## Restore

1. Stop the app and take a safety snapshot of its current dataset.
2. Restore the entire dataset from one snapshot, preserving ownership and permissions. Never mix database/sidecar files from different snapshots.
3. Before exposing the restored app, use the matching image to check data and revoke restored credentials. For the supplied source-checkout Compose deployment:

   ```sh
   docker compose -f deploy/compose.yaml run --rm mindflow check-data
   docker compose -f deploy/compose.yaml run --rm mindflow revoke-credentials
   docker compose -f deploy/compose.yaml up -d
   ```

   On TrueNAS, run equivalent one-off commands against the restored dataset while the normal app is stopped. For a local binary, set `MINDFLOW_DATA=/path/to/data` on each command.
4. Sign in again and verify accounts, workspaces, images, new saves, and SMTP. Resend needed invitations.

`revoke-credentials` removes sessions, invitation/reset links, queued token emails, and editor control; it preserves passwords and content. A restore may resurrect accounts or content deleted after the snapshot. Review those changes before reopening access.

## Application upgrades

Snapshot the data, stop the old container, deploy a pinned new image, and check readiness. For a prebuilt-image Compose deployment, pull the chosen tag before starting it. Startup applies pending domain migrations automatically; there is no separate Goose service.

Each migration is transactional, but an upgrade across domains is not one transaction. A failed migration rolls back its own changes; earlier successful domains remain applied and retry resumes the remaining work. The app refuses newer unsupported schemas. Roll back with the old image **and its matching pre-upgrade snapshot**, rather than assuming an old binary can read the new database.

To migrate explicitly with the supplied Compose file, stop the normal service first, then run:

```sh
docker compose -f deploy/compose.yaml run --rm mindflow migrate
```

The `scratch` image has no shell. Invoke maintenance commands through its entrypoint, or use `/usr/local/bin/mindflow` with `docker exec`.

## Health and troubleshooting

| Symptom/check | What to inspect |
| --- | --- |
| `/health/live` | Process can respond. |
| `/health/ready` | Database is accessible after startup/migrations. An uninitialized instance can be ready for setup. |
| Login works only locally or cookies are missing | Public HTTPS URL, production mode, and reverse-proxy configuration. |
| Email fails | Provider host/port, verified TLS mode, credentials, sender permission, and queued delivery status. |
| Other tabs update slowly | SSE streaming and proxy buffering/caching. Polling is a fallback. |
| Save failed or disconnected | Network, available disk space, permissions, current editor control, and server logs. Preserve the tab until work is saved or recovered. |

SMTP failure does not take the canvas offline. Logs exclude passwords, cookies, token URLs, image bytes, and document bodies, apart from the intentionally logged first-run setup credential.

Image processing is serialized to bound memory. Start with two CPUs and 1 GiB memory, then measure your workload; this is not a guaranteed minimum. Keep free disk for originals, variants, WAL growth, and snapshots. Read the [security policy](../SECURITY.md) before exposing or restoring an instance.
