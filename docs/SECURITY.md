# Security

Mindflow is intended for invite-only deployment behind an HTTPS reverse proxy. Set the public URL to its exact HTTPS origin, keep development mode off, and do not publish the upstream HTTP port to the internet. Cloudflare Tunnel can reach the container's HTTP port privately. Configure edge request limits too: application authentication limits use the direct peer address and deliberately do not trust arbitrary forwarded IP headers.

The production image runs a statically linked binary as UID/GID 10001 in `scratch`, with CA certificates for verified SMTP TLS. It contains no shell or package manager. Mount only the `/data` volume writable; Compose also supplies a temporary filesystem and drops capabilities. Protect database snapshots: they contain account records, private workspaces, images, and SMTP credentials.

The application verifies active sessions, CSRF tokens, request origins, and workspace permissions on the server. Administrators cannot read other users' private workspaces. Passwords use Argon2id; invitation/reset tokens are single-use and expire. SMTP requires verified TLS or STARTTLS. Uploads and request bodies have size limits. Browser content is restricted by CSP and other security headers. Responses containing account or workspace data are not cached. Avatars are profile metadata visible to signed-in users; workspace images require workspace/asset permission.

Follow the [release verification workflow](developer-docs/development.md#verification) before deployment. CI runs quality checks and dependency vulnerability scans on pushes to `main`, pull requests, manual runs, and release verification; published images include provenance and an SBOM. `govulncheck` checks reachable known Go vulnerabilities; npm audit checks known JavaScript dependency advisories. Neither proves the absence of vulnerabilities. Keep dependencies current, review changes, and restore-test your TrueNAS backups.

Report suspected security issues privately to the repository owner, including the affected version and a reproducible example without production credentials or private files.

See [installation](user-docs/installation.md) for HTTPS/container configuration and [operations](user-docs/operations.md) for backup, restore, and upgrade procedures.

PDF originals are opaque protected workspace assets served as attachments. PDF.js renders them locally with bounded file/page/image sizes; the in-app reader has no interactive PDF forms, script actions, or external links. Converted pages are ordinary image assets. PDF parsing remains a browser-side attack surface, so keep PDF.js current with the other dependencies. Passwords accept 8–128 characters; the optional suggester uses two or three randomly selected EFF words and random punctuation from Web Crypto, entirely locally, with suggestions constrained to 10–15 characters.

Authenticated registered users can discover other active profiles through the sharing directory. It exposes display names, account IDs, and avatar references, but no email addresses, account roles, statuses, or credentials. Invited, inactive, and deleted accounts are omitted. Display names are not unique identifiers; sharing remains owner-authorized and keyed by account ID. Profile updates are CSRF-protected and only update the authenticated caller.

Feedback reports are visible only to their author and administrators. Screenshots are optional, previewed before submission, decoded and re-encoded as bounded PNGs, and served through authenticated report access. They may intentionally reveal private canvas content to administrators; they do not grant access to the underlying workspace.

Product-update mail requires explicit opt-in. Unsubscribe uses an unguessable, SHA-256-hashed capability that can only withdraw that consent, without a session. It cannot read data or change credentials. Queued update emails check consent again before delivery. Inactivity deletion excludes administrators and requires successful delivery of both reminders plus the grace periods; authenticated use cancels a pending cycle.
