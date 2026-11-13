# mindflow documentation

mindflow is a self-hosted infinite canvas for images, shapes, text, drawing, and conversations. Start with the section that matches what you need.

## Product

- [Product definition](product/overview.md): who mindflow is for and what it does.
- [Roadmap](product/roadmap.md): planned improvements and deferred features.

## User Docs

- [Installation and setup](user-docs/installation.md): Docker, TrueNAS, HTTPS, email, and initial accounts.
- [Using mindflow](user-docs/using-mindflow.md): canvas controls, sharing, comments, and editor control.
- [Operations](user-docs/operations.md): backups, restore, application upgrades, and troubleshooting.
- [Security](SECURITY.md): deployment boundaries, private data, and vulnerability reporting.

## Developer Docs

- [Project structure](developer-docs/architecture.md): domains, frontend components, and application boundaries.
- [Data structures](developer-docs/data-model.md): database ownership, tables, and canvas JSON.
- [Canvas and synchronization](developer-docs/canvas-and-sync.md): coordinates, saves, editor leases, media, and recovery.
- [Development workflow](developer-docs/development.md): local setup, changes, tests, CI, and releases.
- [Butterfly import](developer-docs/butterfly-import.md): format mapping, limits, and atomic workspace creation.
- [Branding and colors](developer-docs/branding.md): artwork, palette roles, theme accents, and readability.
- [Dependency updates](developer-docs/dependencies.md): `make upgrade` and review steps.
- [Demo recording](developer-docs/demo.md): regenerating the README GIF.
- [Wiki publishing](developer-docs/wiki.md): keeping this documentation synchronized with GitHub.

The Markdown files in [`docs/`](https://github.com/Neokil/mindflow/tree/main/docs) are the source of truth. The wiki publishes them from `main`; edit the repository files rather than the generated wiki copies.
