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
- `go test -race ./cmd/... ./internal/...` passed; frontend type checking/production build and fourteen frontend unit tests passed.
- Browser acceptance suite: thirteen cases across Chromium 158 and Firefox 159, including Firefox device scale factor 2, cross-user sharing/comments and a Chromium insecure-HTTP regression. Covers composition, lasso/grouping, drawing/text/history, persistence, image insertion/transforms/variants, control takeover with disconnected recovery, and canceling an unsaved tab close. Clipboard/drop checks use synthetic native event data rather than the operating system clipboard.
- Latest complete browser run after sharing/comments: all thirteen cases passed in 1.2 minutes, including real invited-user acceptance, role changes, replies/resolution, cross-user takeover and access removal. Large-page frame measurements: Chromium median 16.7 ms / p95 16.8 ms; Firefox median 16.7 ms / p95 33.3 ms in this run.
- Large-page fixture: 3,000 elements, including 200 image placements sharing one source. On this Apple Silicon development machine, observed pan frame medians were approximately 16.7 ms in Chromium and 8–20 ms in Firefox; Firefox tail frames varied around 42–111 ms across runs, including a run concurrent with Linux cross-compilation. This demonstrates basic usability, not a guarantee for 200 unique large images or all hardware. The original 33.3 ms tail-frame goal is not consistently achieved in Firefox and remains a performance optimization target.
- Production Docker image built successfully and passed isolated verification, including running the standalone `migrate` command twice against a fresh volume, with secure HTTPS cookies, SSE streaming through a TLS proxy, invitation delivery over implicit TLS and STARTTLS, owner isolation, original/variant BLOBs, save retries and takeover, non-root/read-only execution, restart persistence, snapshot restore and credential revocation. Fixture SMTP captures messages without contacting a real provider.
- Linux amd64 binary cross-compilation passed. Both Compose YAML files pass `docker compose ... config --quiet`.

## Architecture and automated verification review

- Backend behavior is nested under account and workspace, alongside independent email and installation domains and realtime notification. Only top-level domains own database schemas, repositories, SQL, sqlc bindings and Goose migrations; subdomains use parent storage. The product root contains composition and HTTP protocol mechanics.
- Removed the former database, shared error, identifier and result packages. The product storage package owns connection lifecycle and Goose orchestration. Small local interfaces and explicit transaction arguments connect collaborating domains; no shared repository/service container remains.
- Boundary tests reject horizontal packages, database dependencies outside storage, misplaced migrations/queries, foreign query construction and both reads/writes outside a top-level domain's tables. Mail and image jobs have separate tables; FKs remain within their owning domain.
- Goose 3.30.0 is pinned as a library dependency. Migration packages are discovered by a generator, embedded in the binary and applied with four independent providers/history tables without cross-domain dependencies. CI and Docker check catalog freshness. `mindflow migrate` and startup share this runner.
- Tests verify independent domain upgrades, idempotent repeated runs, transactional migration failure/retry, isolated up/down/up, newer-version refusal and atomic account deletion failure/retry. Development schemas were reset with no legacy upgrade path. A command-level discovery test adds a package and confirms automatic linking and stale-catalog rejection.
- Account credential changes revoke account-owned sessions/tokens and cancel mail through a collaborator. Account deletion explicitly coordinates workspace cleanup and queued email in the same transaction. Stopped-instance restore additionally invalidates editor control in the same transaction. Authentication owns request/session/CSRF and administrator middleware.
- Existing account-policy, credential-serialization, editor-fencing, image-orientation and integration tests live with their owning domains. The HTTP/API behavior remains compatible.
- Frontend components are split by accounts, workspaces and editor responsibilities. Scene rendering, elements, text entry, controls, inspector and shortcuts are separate components/modules. Typed props and callbacks replace the previous global custom-event bridge.
- Credential serialization is explicitly excluded and tested. SMTP settings responses redact passwords. Infrastructure failures retain server-error semantics rather than being misreported as invalid credentials.
- `make lint` checks sqlc output freshness, golangci-lint 2.15.0 with all linters enabled except the twelve requested exclusions, gofumpt/goimports formatting, ESLint/React hooks, Prettier and Python syntax. Configuration validation passes. Builds enforce TypeScript unused-code checks. Domain/boundary tests complement existing integration tests.
- `make lint test` passes: golangci-lint reports zero issues, all Go tests pass under the race detector, and all fourteen frontend unit tests pass. `sqlc diff` and catalog freshness checks confirm generated code matches the committed migrations and SQL.
- GitHub Actions defines quality, Chrome/Firefox acceptance and production-container jobs for pushes and pull requests. A dependent publishing job pushes Linux amd64/arm64 images to `neokil/mindflow` for main/version tags, using the Docker Hub repository secrets. Action versions are pinned to commit hashes. The workflow passes local actionlint validation; hosted execution requires pushing the commits.

## Operational checks requiring the deployment environment

The application and tests are implemented; these checks are not represented as completed:

- SMTP delivery using the operator's real provider; automated tests do not send real email.
- Cloudflare Tunnel HTTPS cookies, streaming/caching configuration, and origin reachability on the actual network.
- TrueNAS installation, dataset permissions, snapshot/restore drill, and resource measurement with the owner's image workload.

See [RUNNING.md](RUNNING.md) for exact setup and operating steps. No public image or production instance has been published.

## HTTP UUID regression

The original workspace acceptance suite used `127.0.0.1`, a browser-trusted loopback origin. It missed that `crypto.randomUUID()` is unavailable over ordinary HTTP hostnames. A real `http://mindflow.test:8087` Chromium test reproduced the reported blank workspace and exact exception with the previous implementation. Frontend IDs now use `crypto.getRandomValues()` and UUID v4 formatting, with no randomUUID requirement. A unit test verifies the version/variant with randomUUID absent. The browser regression checks the origin is actually insecure and randomUUID absent before opening, editing, saving and reopening the workspace.

Validation: the previous implementation failed the real HTTP regression with `crypto.randomUUID is not a function`; the fixed build passed all nine Playwright cases in 35.8 seconds, plus ten unit tests, ESLint, formatting and the TypeScript/production build.

## Element editing improvements

Move cursors reflect selectable/movable elements and active dragging, preserving tool and transform cursors. Text font-size inputs accept whole numbers from 1 to 2,000; corner transforms round the resulting size. Vertical alignment persists as top/middle/bottom and uses matching wrapped-text metrics in the textarea overlay.

A separate crop dialog provides rectangle drawing, moving, corner handles and percentage trims. Apply is one history edit; Cancel discards the draft, and Reset restores the original bounds. Normalized crop coordinates preserve the original BLOB and work with image variants, rotated placements, duplication, undo/redo and persisted documents. Preview resolution accounts for the cropped fraction.

Validation: `make lint test` passed, with zero golangci-lint issues and fourteen frontend unit tests. All eleven Playwright cases passed on the final code. The new cases check hover/drag cursors, integer inputs and resize results, rendered text placement, crop persistence, undo/redo, canceled reset and restored image bounds. Crop UI screenshots were reviewed.

## Workspace sharing and comments

- Workspace migration 2 preserves existing private workspaces, documents and image BLOBs. Membership, comment threads/messages, access views and lease holder identities stay in workspace storage, with no cross-domain tables or FKs. An upgrade integration test verifies preserved content and private defaults.
- SharingDialog manages existing active users by email with Read-only, Comment and Full access roles. Only the owner manages grants or workspace lifecycle. Shared workspaces appear in members' lists. Role changes fence only the affected user's leases.
- CommentsLayer provides positioned initials, hover/focus threads, replies, creator/owner resolution/reopening/deletion and resolved-thread visibility. Commenting is independent of canvas edit control and document revision. Server-assigned authors and plain-text rendering prevent attribution forgery and HTML execution.
- Shared image uploads and preview jobs belong to the workspace owner. Access checks include originals/variants and reject unrelated private images or foreign asset attachment. Deleting a collaborator removes their memberships/authored discussion and held leases without removing retained shared uploads. Pending recovery uploads use the chosen target workspace.
- Go API tests cover role restrictions, owner-only sharing, private image isolation, lease takeover/stale writes, comment management, resolved reply refusal, revocation and collaborator deletion. The two browser cases use real invited accounts, then exercise sharing, comments, takeover and access removal in Chrome and Firefox. Invitation tokens are read only from the disposable fixture database; no production test endpoint exists.
- Verification passed: all thirteen browser cases; fourteen frontend unit tests; Go race tests; golangci-lint, frontend lint/format/build and generated SQL checks; production-container migration, HTTPS/SSE/mail and snapshot/restore checks. Workflow validation passes locally. Screenshots of shared canvases were reviewed.
- Two earlier Chromium runs paused browser activity long enough to time out autosave assertions, then completed the saves. The final complete run passed with unchanged timeouts. Failed-run traces are retained locally under `.cache/sharing-artifacts-attempt1` and `attempt2` for further investigation if it recurs.
