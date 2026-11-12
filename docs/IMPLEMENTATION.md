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

Go 1.26.11; SQLite through modernc.org/sqlite 1.62.0; golang.org/x/crypto 0.59.0 and x/image 0.48.0; React 19.3.1; TypeScript 5.9.3; Vite 7.3.8; Konva 10.3.0 / react-konva 19.2.3. Konva was adopted after the interaction prototype and browser checks. Runtime license notices are in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).

## Validation — 12 November 2026

- Go integration/service tests use real temporary SQLite databases: setup, invitations/reset, session/CSRF/origin rules, last-administrator protection, private content/assets, duplication, deletion, save retries/revisions, lease fencing, image variants/orientation, and restore maintenance.
- `go test -race ./...` passed; frontend type checking/production build and nine geometry/save-state unit tests passed.
- Browser acceptance suite: eight scenarios across Chromium 158 and Firefox 159, including Firefox device scale factor 2. Covers composition, lasso/grouping, drawing/text/history, persistence, image insertion/transforms/variants, control takeover with disconnected recovery, and canceling an unsaved tab close. Clipboard/drop checks use synthetic native event data rather than the operating system clipboard.
- Large-page fixture: 3,000 elements, including 200 image placements sharing one source. On this Apple Silicon development machine, observed pan frame medians were approximately 16.7 ms in Chromium and 8–20 ms in Firefox; Firefox tail frames varied around 42–111 ms across runs, including a run concurrent with Linux cross-compilation. This demonstrates basic usability, not a guarantee for 200 unique large images or all hardware. The original 33.3 ms tail-frame goal is not consistently achieved in Firefox and remains a performance optimization target.
- Linux amd64 binary cross-compilation passed. Both Compose YAML files pass `docker compose ... config --quiet`.

## Operational checks requiring the deployment environment

The application and tests are implemented; these checks are not represented as completed:

- Container build/start on the target host: the local Docker daemon was unavailable.
- SMTP delivery using the operator's real provider; automated tests do not send real email.
- Cloudflare Tunnel HTTPS cookies, streaming/caching configuration, and origin reachability on the actual network.
- TrueNAS installation, dataset permissions, snapshot/restore drill, and resource measurement with the owner's image workload.

See [RUNNING.md](RUNNING.md) for exact setup and operating steps. No public image or production instance has been published.
