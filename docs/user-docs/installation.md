# Installation and setup

Run one mindflow instance per database. The application serves HTTP on port 8080; browsers should reach it through your HTTPS reverse proxy or Cloudflare Tunnel. All persistent data, including images, lives in `/data`.

## Docker Compose

Use the Compose example in the [README](../../README.md#installation) for a prebuilt image. From a source checkout you can build and start the supplied deployment:

```sh
docker compose -f deploy/compose.yaml up --build -d
docker compose -f deploy/compose.yaml logs mindflow
```

The container runs as UID/GID `10001:10001` in `scratch`, with a read-only root filesystem, a writable data volume, and temporary `/tmp`. Its health check is built into the application. Use a version tag or digest for stable deployments; `neokil/mindflow:edge` follows the main branch. Availability of a tag depends on a successful image publishing run.

## HTTPS and network configuration

Route your public hostname to the service's private HTTP address. Do not expose the upstream port directly to the internet. If the tunnel runs in another container, `localhost` refers to that container; connect it through a suitable private Docker network or host address.

Bypass proxy caching for `/api/*`, including protected images and events. Verify that server-sent events stream through your proxy; polling provides a fallback. Public URLs and secure cookies require HTTPS in production. `MINDFLOW_DEV=1` permits local HTTP for development and should remain unset on an exposed installation.

| Environment variable | Default | Meaning |
| --- | --- | --- |
| `MINDFLOW_ADDR` | `:8080` | HTTP bind address. |
| `MINDFLOW_DATA` | `data`; `/data` in the container | Persistent data directory. |
| `MINDFLOW_DEV` | Unset | Enable local HTTP development mode when set to `1`. |

The public application URL comes from setup. Forwarded headers are not trusted for authentication or client-IP limits; users behind one tunnel share the application's authentication rate limit. Configure edge limits as appropriate.

## TrueNAS Apps

1. Create a local dataset, such as `/mnt/POOL/apps/mindflow`, and grant UID/GID `10001:10001` read/write and traversal permissions. Do not host the SQLite database on SMB/NFS.
2. Edit [the TrueNAS YAML](../../deploy/truenas.yaml), choosing an image tag, dataset path, and host port.
3. In **Apps → Discover Apps → three-dot menu → Custom App → Install via YAML**, paste the edited configuration.
4. Watch the `migrate` service: it must finish with exit code 0 before `mindflow` starts. A completed migration container is expected to remain stopped. Read the `mindflow` log for the first-run setup credential, route your HTTPS hostname, and complete setup.
5. Verify login, email, saving across restart, and dataset snapshot/restore on your installation.

The TrueNAS YAML uses a one-shot `migrate` service with `depends_on: condition: service_completed_successfully`. It runs the image's `migrate` command with the same data mount and UID/GID as the server. It has no network access, published port, or HTTP health check. The server also checks pending migrations on startup, so ordinary container restarts remain safe.

`MINDFLOW_DATA` is a directory, not a database filename. With `/data`, the app creates `/data/mindflow.sqlite` automatically; do not create an empty database file manually. Both containers need write access to the dataset and its SQLite/WAL files. The supplied `create_host_path: false` deliberately requires the host directory to exist. A migration service cannot fix a missing mount, read-only dataset, or insufficient TrueNAS ACL permissions. Grant UID/GID `10001:10001` access through the dataset permissions/ACL editor, including traversal where needed, while retaining your existing ACL entries.

Images are built for Linux amd64 and arm64. Use the appropriate architecture for your server.

## Initial setup and email

The server logs a random setup credential while its database is uninitialized. Copy it into the setup screen, then enter the first administrator's email/password, the exact public HTTPS origin, and SMTP settings. The public URL must have no extra path, query, or fragment. The setup credential changes on restart until setup completes; treat log access as operator access.

SMTP requires certificate-verified implicit TLS or STARTTLS. Common provider ports are 465 and 587 respectively; use your provider's actual settings and a permitted sender address. Send a test email to confirm delivery. The app does not configure DNS or the email provider for you.

## Accounts and administration

Administrators invite accounts by email; there is no public registration. They can resend invitations, send password resets, change roles, deactivate/reactivate users, and delete accounts. At least one active administrator must remain. Administration does not grant access to other users' workspaces. Deleting an account permanently removes its owned workspaces and related account data.

Passwords contain 15–128 characters. Invitations expire after seven days, reset links after one hour, and sessions after 30 days unless revoked earlier. Resending a link supersedes the old one. Email is queued and retried; user management shows delivery failures. Fix SMTP settings and resend if delivery has failed permanently. A blank SMTP password during settings updates retains the stored secret.

Continue with [Using mindflow](using-mindflow.md), [Operations](operations.md), and the [security policy](../SECURITY.md).
