# New project standard

Use this baseline when creating a project. Deliver a working feature through the UI, API, database, tests, and production container. Keep the design proportionate to the product; document justified exceptions.

## Stack and structure

- **Backend:** Go, sqlc, Goose, and a SQL database chosen for deployment needs.
- **Frontend:** React, strict TypeScript, Vite, Vitest, and Playwright.
- **Delivery:** Docker, GitHub Actions, and MIT licensing.
- Keep frontend and backend source separate. They may ship together as one static Go binary serving the frontend and API.

```text
cmd/<app>/                    # Entrypoint and dependency composition
internal/<domain>/            # Business rules, services, HTTP adapters
  storage/                    # Repository adapters and generated sqlc code
    queries/                  # Domain-owned SQL
    migrations/               # Goose migrations
web/
  src/features/               # Feature components, hooks, and tests
  src/components/             # Shared UI components
  src/api/                    # Typed API client
  e2e/                        # Playwright tests and fixtures
api/openapi.yaml              # HTTP contract
.github/workflows/            # CI, release, and wiki workflows
scripts/                      # Generation and maintenance tools
tests/container/              # Production runtime verification
deploy/                       # Hardened Compose example
docs/                         # Product, user, and developer documentation
AGENTS.md                     # Agent rules, boundaries, and check commands
README.md                     # Purpose, quickstart, and documentation links
LICENSE                       # MIT license
THIRD_PARTY_NOTICES.md         # Required dependency and asset notices
Makefile                      # Stable development commands
Dockerfile                    # Multi-stage build ending in scratch
```

## Architecture and API

- Use domain-driven design: organize backend behavior by business capability, with explicit ownership and small interfaces between domains.
- Keep business rules independent of HTTP, SQL, and generated types. Handlers translate requests; services enforce rules; storage adapters execute queries.
- Each top-level domain owns its schema, queries, and migrations. Subdomains share parent storage. Collaborate through interfaces rather than accessing another domain's tables.
- Define transaction boundaries. Use cancellation, timeouts, bounded background work, and graceful shutdown. Make retried operations safe where needed.
- Organize frontend code by feature with focused components and clear state ownership. Centralize API access and shared persistence behavior.
- Maintain OpenAPI alongside implementation changes, including authentication, schemas, errors, and pagination. Generate frontend types/client bindings or enforce equivalent contract checks. Validate the spec and representative responses in CI.

## Data and security

- Use parameterized sqlc queries. Commit SQL, generated bindings, and append-only Goose migrations together; check generated-code freshness in CI.
- Never edit applied migrations. Test fresh installation and upgrades with existing data. Document destructive changes, backup/restore, and rollback limitations.
- Enforce authorization and input validation on the backend. Bound uploads, requests, expensive processing, and authentication attempts where applicable.
- Require HTTPS in production, verify outbound TLS, and apply appropriate cookie, CSRF, CORS, and browser security controls.
- Load secrets at runtime; exclude them from source, images, browser bundles, and logs. Document configuration with safe example values.
- Define sensitive-data retention/deletion rules and private vulnerability reporting. Automate dependency updates and vulnerability checks.

## Production container

- Use a multi-stage Dockerfile ending in `FROM scratch`, with a static Go binary built using `CGO_ENABLED=0` and compatible dependencies.
- Include only required runtime files, certificates, assets, and license notices. Use a fixed numeric non-root UID/GID and an exec-form entrypoint.
- Provide a read-only root filesystem, explicitly writable data/temp mounts, dropped capabilities, and `no-new-privileges` in deployment examples.
- Pin and maintain build inputs, including base-image digests. Minimize build context with `.dockerignore`; keep build secrets out of layers.
- Verify startup, health checks, persistence, shutdown, and restore using the actual production image. Implement health checks without requiring a shell.

## Tests and developer workflow

- Include Go unit tests for business rules, frontend unit/component tests, real-database integration tests, API contract tests, and Playwright E2E tests for critical journeys and failures.
- Use isolated, deterministic fixtures with disposable data. Never depend on production accounts or external side effects. Retain browser failure artifacts.
- Run formatting, lint, TypeScript checks, generated-code checks, dependency audits, and relevant tests in CI; use Go's race detector where supported.
- Pin compatible tool versions and commit lockfiles. Document setup from a clean clone and expose consistent commands for setup, development, build, generation, lint, tests, audits, and container verification.
- Support loading, empty, error, and recovery states. Provide keyboard access, visible focus, accessible labels, and responsive layouts; declare supported browsers.

## Documentation and wiki

- Keep Markdown under `docs/` in the source repository as the source of truth; publish it to the GitHub wiki. Update documentation with code changes.
- Document product purpose, non-goals, acceptance criteria, architecture, setup, configuration, usage, operations, and release procedures. Record consequential decisions briefly.
- Publish the release commit's docs after successful artifact publication, showing version and source commit. Support manual publication of a selected revision and local preview.
- Preserve links/assets, remove stale managed pages, preserve unrelated wiki pages, serialize publishing, and avoid commits when nothing changed.
- Keep `AGENTS.md` concise. Include architecture boundaries, generated-file rules, and verification commands. Preserve required third-party licenses and asset notices alongside the MIT license.

## CI and publishing

Use focused reusable workflows:

| Workflow | Responsibility |
| --- | --- |
| `ci.yml` | Coordinate checks for pull requests, main pushes, and release verification. |
| `quality.yml` | Formatting, lint, types, unit/integration/contract tests, generation checks, and audits. |
| `browsers.yml` | Playwright tests for supported browsers and failure artifacts. |
| `container.yml` | Production image, runtime, persistence, and recovery checks. |
| `release.yml` | Verify the exact release commit, publish versioned artifacts, then publish docs. |
| `wiki.yml` | Synchronize documentation for the selected revision. |

Pin third-party Actions to full commit SHAs. Use minimal token permissions, explicit timeouts, and appropriate concurrency. Expose publishing credentials only to trusted publishing jobs. Configure required branch checks where available.

Publish on a published GitHub release after checks pass. Publish the verified image, with supported architectures, SBOM, and provenance. Stable releases advance `latest`; prereleases do not. Never replace a version with different contents. Document release setup and retry procedures.

## Completion checklist

- [ ] One useful feature works through all layers, with meaningful tests passing.
- [ ] A clean clone can run, build, and execute documented checks.
- [ ] The hardened scratch image runs correctly; migration and restore paths are tested.
- [ ] Security controls, configuration, health checks, structured logs, and backup procedures are in place.
- [ ] Documentation, wiki publishing, CI/release workflows, and license notices are complete.
- [ ] Handoff lists executed checks, known limitations, and any external settings or credentials still required. Report only verification actually performed.
