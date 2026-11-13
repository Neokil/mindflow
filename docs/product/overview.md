# Product definition

mindflow provides a quiet, flexible space for collecting images, arranging thoughts, sketching, and discussing ideas. It is a self-hosted desktop web application for an individual and a small group of invited users, with an interface inspired by Apple applications.

## What it does

Users organize their work in private-by-default workspaces containing tabbed infinite-canvas pages. Pages support images, rectangle/circle/star shapes, plain text with element-wide formatting, and freehand pen strokes. Elements can be selected, moved, resized, rotated, grouped, duplicated, cropped where applicable, and arranged in layers.

Butterfly version-13 exports can be imported into new private workspaces, preserving ordered pages, embedded images, text, freehand strokes, and rectangle/ellipse shapes. Conversion notes identify approximations and unsupported content.

Workspaces can be shared with existing users as read-only, comment, or full access. Comment threads attach conversations to canvas points. Each page has one writer at a time; other users can request control, and the workspace owner can force a transfer. Saved changes are visible to other sessions.

Accounts are invite-only. Administrators manage users and invitations without gaining access to private workspaces. The first administrator and email settings are configured during initial setup.

## Product boundaries

- Desktop Chrome and Firefox are the supported browser targets.
- One application instance serves a SQLite database on the operator's server; TrueNAS is a deployment option.
- Accounts, documents, original images, variants, and settings are stored on the server.
- Loaded pages can be edited temporarily while disconnected, but unsaved drafts live only in browser-session memory. This is not a persistent offline application.
- Sharing requires an existing account. A page URL is a destination, not a public-access grant.
- Collaboration uses one writer per page, with comments available independently of editor control.

See the [user guide](../user-docs/using-mindflow.md) for current behavior and the [roadmap](roadmap.md) for features that are not yet available.
