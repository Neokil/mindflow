# Implementation milestones

Each completed milestone is committed separately. This checklist tracks implementation, not a change in product scope.

- [x] Foundation: Go/SQLite application, React build, health checks, development workflow.
- [x] Identity: first-run setup, invitations, login/reset, administration, email delivery.
- [x] Persistence: private workspaces/pages, assets/variants, revisioned saving, editor leases/events.
- [x] Editor: canvas navigation, insertion, selection, transforms, drawing, text, history.
- [x] Integration: autosave, disconnected editing, read-only updates/takeover/recovery, settings.
- [x] Delivery: container/TrueNAS manifests, operator instructions, focused acceptance tests.

Validation records and remaining operational checks will be recorded here as implementation progresses.

## Implemented stack

Go 1.26.11; Goose 3.30.0; SQLite through modernc.org/sqlite 1.62.0; golang.org/x/crypto 0.59.0 and x/image 0.48.0; React 19.3.1; TypeScript 5.9.3; Vite 7.3.8; Konva 10.3.0 / react-konva 19.2.3. Konva was adopted after the interaction prototype and browser checks. Runtime license notices are in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).

## Validation — 12 November 2026

- Go integration/service tests use real temporary SQLite databases: setup, invitations/reset, session/CSRF/origin rules, last-administrator protection, private content/assets, duplication, deletion, save retries/revisions, lease fencing, image variants/orientation, and restore maintenance.
- `go test -race ./cmd/... ./internal/...` passed; frontend type checking/production build and nine geometry/save-state unit tests passed.
- Browser acceptance suite: eight scenarios across Chromium 158 and Firefox 159, including Firefox device scale factor 2. Covers composition, lasso/grouping, drawing/text/history, persistence, image insertion/transforms/variants, control takeover with disconnected recovery, and canceling an unsaved tab close. Clipboard/drop checks use synthetic native event data rather than the operating system clipboard.
- The browser suite passed again after the domain-storage and Goose changes. Earlier measured run: all eight tests passed in 46.0 seconds. Large-page frame measurements: Chromium median 16.7 ms / p95 16.8 ms; Firefox median 7.5 ms / p95 62.9 ms.
- Large-page fixture: 3,000 elements, including 200 image placements sharing one source. On this Apple Silicon development machine, observed pan frame medians were approximately 16.7 ms in Chromium and 8–20 ms in Firefox; Firefox tail frames varied around 42–111 ms across runs, including a run concurrent with Linux cross-compilation. This demonstrates basic usability, not a guarantee for 200 unique large images or all hardware. The original 33.3 ms tail-frame goal is not consistently achieved in Firefox and remains a performance optimization target.
- Production Docker image built successfully and passed isolated verification, including running the standalone `migrate` command twice against a fresh volume, with secure HTTPS cookies, SSE streaming through a TLS proxy, invitation delivery over implicit TLS and STARTTLS, owner isolation, original/variant BLOBs, save retries and takeover, non-root/read-only execution, restart persistence, snapshot restore and credential revocation. Fixture SMTP captures messages without contacting a real provider.
- Linux amd64 binary cross-compilation passed. Both Compose YAML files pass `docker compose ... config --quiet`.

## Architecture and automated verification review

- Backend now has separate account, session, authentication, invitation, password-reset, email, installation, workspace, page, canvas, editor, image and realtime-notification packages. Each owns its behavior and controllers; all database code, repositories, SQL, sqlc bindings and Goose migrations are in its `storage` subpackage. The product root contains composition and HTTP protocol mechanics.
- Removed the former database, shared error, identifier and result packages. The product storage package owns connection lifecycle and Goose orchestration. Small local interfaces and explicit transaction arguments connect collaborating domains; no shared repository/service container remains.
- Boundary tests reject horizontal packages, database dependencies outside storage, misplaced migrations/queries, foreign query construction and writes outside a domain's tables/row partition. Legacy token/job rows retain explicit purpose/kind guards.
- Goose 3.30.0 is pinned as a library dependency. Migration packages are discovered by a generator, embedded in the binary and applied with separate providers/history tables in dependency order. CI and Docker check catalog freshness. `mindflow migrate` and startup share this runner.
- Tests verify independent domain upgrades, idempotent repeated runs, transactional migration failure/retry, legacy content/BLOB preservation, newer-version refusal and missing/cyclic dependencies. A command-level discovery test adds a package and confirms automatic linking and stale-catalog rejection.
- Account credential changes coordinate revocation through sessions, invitations, password resets and queued email. Stopped-instance restore additionally invalidates editor control in the same transaction. Authentication owns request/session/CSRF and administrator middleware.
- Existing account-policy, credential-serialization, editor-fencing, image-orientation and integration tests live with their owning domains. The HTTP/API behavior remains compatible.
- Frontend components are split by accounts, workspaces and editor responsibilities. Scene rendering, elements, text entry, controls, inspector and shortcuts are separate components/modules. Typed props and callbacks replace the previous global custom-event bridge.
- Credential serialization is explicitly excluded and tested. SMTP settings responses redact passwords. Infrastructure failures retain server-error semantics rather than being misreported as invalid credentials.
- `make lint` checks sqlc output freshness, golangci-lint 2.15.0 with all linters enabled except the twelve requested exclusions, gofumpt/goimports formatting, ESLint/React hooks, Prettier and Python syntax. Configuration validation passes. Builds enforce TypeScript unused-code checks. Domain/boundary tests complement existing integration tests.
- `make lint test` passes: golangci-lint reports zero issues, all Go tests pass under the race detector, and all nine frontend unit tests pass. `sqlc diff` and catalog freshness checks confirm generated code matches the committed migrations and SQL.
- GitHub Actions defines quality, Chrome/Firefox acceptance and production-container jobs for pushes and pull requests. Action versions are pinned to commit hashes. The workflow passes local actionlint validation; hosted execution requires pushing the commits.

## Operational checks requiring the deployment environment

The application and tests are implemented; these checks are not represented as completed:

- SMTP delivery using the operator's real provider; automated tests do not send real email.
- Cloudflare Tunnel HTTPS cookies, streaming/caching configuration, and origin reachability on the actual network.
- TrueNAS installation, dataset permissions, snapshot/restore drill, and resource measurement with the owner's image workload.

See [RUNNING.md](RUNNING.md) for exact setup and operating steps. No public image or production instance has been published.
