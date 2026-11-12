# mindflow

A minimalistic, self-hosted infinite canvas web app for images, shapes, text, and freehand drawing, with an interface inspired by Apple applications.

## Planned first release

- Invite-only email/password accounts and user administration.
- Private workspaces with tabbed canvas pages.
- Infinite panning, cursor-centered zoom, and freehand lasso selection.
- Image upload, drag-and-drop, and clipboard paste.
- Circles, rectangles, stars, basic formatted text, and mouse drawing.
- Element transforms, grouping, undo/redo, and optional snap-to-grid.
- Autosave, temporary disconnected editing, and light/dark modes.

Intended for desktop browsers and hosting on a private server, preferably TrueNAS.

## Status

Product requirements and technical design drafts are documented. Implementation has not started. The planned stack is Go, React/TypeScript with Vite, and SQLite storing all application data, including images. Konva will be evaluated in an early editor prototype.

Setup and development instructions will be added with the implementation.

See [REQUIREMENTS.md](docs/REQUIREMENTS.md) for agreed behavior, acceptance criteria, and deferred features, including workspace sharing, export, and Linwood Butterfly import.

Technical drafts:

- [Architecture](docs/ARCHITECTURE.md): components, data model, API, authentication, and authorization.
- [Canvas and synchronization](docs/CANVAS_AND_SYNC.md): interactions, image variants, autosave, editor takeover, and recovery.
- [Deployment and operations](docs/DEPLOYMENT.md): TrueNAS, Cloudflare Tunnel, SQLite snapshots, upgrades, and verification.
