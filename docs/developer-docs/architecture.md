# Project structure

mindflow is one Go process serving an embedded React/TypeScript frontend, a same-origin HTTP API, background workers, and SQLite. It runs one instance per database. The production image uses a static binary in `scratch`; there is no separate database service.

## Repository map

| Location                                      | Responsibility                                                                                                                                   |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `cmd/mindflow`                                | Server entrypoint, health check, migrations, data checks, and credential revocation commands.                                                    |
| `cmd/genmigrations`                           | Generate/check the embedded domain migration catalog.                                                                                            |
| `internal/account`                            | Accounts, password policy, avatars, authentication, sessions, invitations, and resets.                                                           |
| `internal/workspace`                          | Workspaces, sharing, pages, canvas documents, editor control, comments, images, imports, and workspace templates.                                |
| `internal/email`                              | SMTP delivery and retry queue.                                                                                                                   |
| `internal/installation`                       | Initial setup and instance configuration.                                                                                                        |
| `internal/mindflow`                           | Compose domain services and provide common HTTP mechanics. Its storage package owns connection/migration infrastructure, not all domain queries. |
| `internal/notification`                       | Authenticated SSE invalidations.                                                                                                                 |
| `web/src/features`                            | Account, workspace, and editor UI components.                                                                                                    |
| `web/src/session.ts`                          | Per-page loading, save generations, lease checks, reconnection, and recovery.                                                                    |
| `web/src/model.ts`, `geometry.ts`, `media.ts` | Canvas types, coordinate/selection math, and image loading.                                                                                      |
| `api/openapi.yaml`                            | HTTP contract.                                                                                                                                   |
| `deploy`, `.github/workflows`, `scripts`      | Deployment examples, CI/publishing, and maintenance tools.                                                                                       |

## Backend domains

Behavior belongs to the domain that owns it. Account subdomains include `authentication`, `session`, `invitation`, `passwordreset`, and `avatar`. Workspace subdomains include `sharing`, `image`, `template`, `transfer`, `butterfly`, `page`, and the page's `canvas`, `editor`, and `comment` packages.

Only the topmost domain defines its database structure. Its `storage/` contains repository adapters, sqlc queries/generated bindings, and Goose migrations. Nested domains share that parent storage. The account, workspace, email, installation, release, and feedback schemas are independent: no cross-domain foreign keys or table access. References into another domain are opaque IDs.

Domains collaborate through explicit service interfaces. For example, sharing asks the account service to resolve an active user; it does not query account tables. Account deletion coordinates content and credential cleanup through interfaces. A shared transaction handle supports atomic collaboration without granting cross-domain table access.

Authentication is a domain and HTTP middleware. Domain handlers declare whether an endpoint requires a session or administrator; business services still enforce workspace roles and resource access. Administrators have no implicit content access.

## Frontend boundaries

React owns application navigation and UI state. Konva renders a viewport-sized canvas; world coordinates remain independent of the camera. `CanvasScene` renders/transforms elements, while components handle tools, text input, crop dialogs, comments, canvas settings, and anchored object properties. Canvas and object inspectors have separate visibility state. `TemplateLibrary` manages named snapshots and `templates.ts` creates independent placements; template persistence belongs to the workspace `template` subdomain and parent storage.

`PageSession` owns the saved document/revision, client identity, editor epoch, local history, pending saves, and recovery copy. Components request edits instead of implementing parallel save protocols. Media loading keeps original/variant selection separate from document editing.

Workspace/page hash URLs are handled by `navigation.ts`; `UpdateBanner` checks the running build. Both preserve the existing unsaved-work protections. SSE carries invalidations, not authoritative document state; HTTP and polling remain the source of truth.

Read [data structures](data-model.md), [canvas and synchronization](canvas-and-sync.md), and the [development workflow](development.md) before changing these boundaries.

Native transfer lives in `workspace/transfer`; its storage stays in the parent domain. Frontend `features/transfer` owns portable-file UI and visual exports, sharing the editor element renderer. See [native import and export](native-transfer.md).

`WorkspacePreview` lazily renders the first saved page on each workspace card using the shared canvas export renderer. The workspace list returns `firstPageId` and its `previewRevision` from the authorized workspace query; no thumbnail tables or new migrations are needed. Page invalidations refresh that metadata. Only one thumbnail renders at a time, and generated object URLs are revoked when a card changes or unmounts.

`workspace/trash` owns soft deletion recovery and expiry; `workspace/ownership` owns ownership transfer. Their SQL, migrations, and generated sqlc bindings live in `workspace/storage`. Canvas links are validated by `workspace/page/canvas` and stored in page JSON; the frontend link picker and navigation remain separate from shape rendering. Alignment is a pure editor operation shared by the selection toolbar and object inspector. Transactional welcome notifications use the email outbox without introducing an account-to-email storage dependency.

## Account lifecycle, releases and feedback

`account/inactivity` advances delivery-aware 90/30/7-day reminders using `account/storage`. Authentication records activity with hourly write throttling and immediately resets any pending deletion cycle. The worker rechecks role, status and activity within the deletion transaction. Administrators are exempt. Email owns successful SMTP-delivery receipts; missing receipts prevent deletion. Root composition coordinates workspace and feedback cleanup with account deletion atomically.

`account/updates` owns explicit consent and hashed, purpose-limited unsubscribe capabilities. `release` owns a highest-observed-version cursor in independent storage. Upgrading queues subscriber mail and advances the cursor in one transaction; first installation establishes a baseline, while restarts, development builds and downgrades do nothing. This is a durable outbox, not exactly-once SMTP: a process crash after SMTP acceptance can cause a delivery retry. Consent is checked again before sending. No external release lookup is required.

`feedback` owns reports, discussion, screenshot BLOBs and admin decisions. Opaque author IDs have no account foreign keys. Domain queries authorize author/admin access before returning report content. Screenshots are bounded, decoded and re-encoded as PNG before persistence. The frontend separates lists, composition, discussion/review and lazy-loaded screenshot capture. Only explicitly submitted screenshots grant administrators a view of otherwise private content.

Feedback lifecycle rules live in the feedback domain's pure `Decide` policy and are enforced again against transactionally loaded state before writes. The API returns the current allowed actions. `Reports` owns the always-visible list and nested creation/detail dialogs; `ReportThread` owns discussion and action submission; `useFeedbackUnread` polls per-user counts and refreshes after read/activity acknowledgements. No workspace access is granted by a report or its read cursor.

Report attachments use a single browser form with a type selector. `imageAttachment` decodes browser-supported formats, checks file/pixel limits, and converts them to static PNGs; the server independently decodes and normalizes the PNG. Dialog cancellation handles only events targeting the dialog itself, so a file picker's bubbling cancel event cannot dismiss its parent report. Feedback owns both the closed-report expiry policy and authenticated content search; neither introduces another domain's table access.
