# Security

Mindflow is intended for invite-only deployment behind an HTTPS reverse proxy. Set the public URL to its exact HTTPS origin, keep development mode off, and do not publish the upstream HTTP port to the internet. Cloudflare Tunnel can reach the container's HTTP port privately. Configure edge request limits too: application authentication limits use the direct peer address and deliberately do not trust arbitrary forwarded IP headers.

The production image runs a statically linked binary as UID/GID 10001 in `scratch`, with CA certificates for verified SMTP TLS. It contains no shell or package manager. Mount only the `/data` volume writable; Compose also supplies a temporary filesystem and drops capabilities. Protect database snapshots: they contain account records, private workspaces, images, and SMTP credentials.

The application verifies active sessions, CSRF tokens, request origins, and workspace permissions on the server. Administrators cannot read other users' private workspaces. Passwords use Argon2id; invitation/reset tokens are single-use and expire. SMTP requires verified TLS or STARTTLS. Uploads and request bodies have size limits. Browser content is restricted by CSP and other security headers. Responses containing account or workspace data are not cached.

Run `make lint test audit` and `make test-e2e verify-container` before deployment. CI runs these checks and dependency vulnerability scans; published images include provenance and an SBOM. `govulncheck` checks reachable known Go vulnerabilities; npm audit checks known JavaScript dependency advisories. Neither proves the absence of vulnerabilities. Keep dependencies current, review changes, and restore-test your TrueNAS backups.

Report suspected security issues privately to the repository owner, including the affected version and a reproducible example without production credentials or private files.
