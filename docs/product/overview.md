# Product definition

mindflow provides a quiet, flexible space for collecting images, arranging thoughts, sketching, and discussing ideas. It is a self-hosted web application for individuals and small groups of invited users, with desktop and tablet support and an interface inspired by Apple applications.

## What it does

Users organize their work in private-by-default workspaces containing tabbed infinite-canvas pages. Pages support images, rectangle/circle/star shapes, plain text with element-wide formatting, and freehand pen strokes with optional Pencil pressure and adjustable stabilization. Elements can be selected, moved, resized, rotated, grouped, duplicated, cropped where applicable, aligned, linked to other elements on any page of the workspace, and arranged in layers.

Butterfly version-13 exports can be imported into new private workspaces, preserving ordered pages, embedded images, text, freehand strokes, and rectangle/ellipse shapes. Conversion notes identify approximations and unsupported content.

Selections can be saved as workspace templates and inserted on other pages as independently editable copies. Groups can be locked to protect them against accidental editing or movement.

Workspaces can be shared with existing users as read-only, comment, or full access. Comment threads attach conversations to canvas points. Each page has one writer at a time; other users can request control, and the workspace owner can force a transfer. Saved changes are visible to other sessions. Owners can transfer a workspace to another active user and retain full access. Deleted workspaces/pages can be restored from Trash for 30 days before permanent removal.

Accounts are invite-only. Administrators manage users and invitations without gaining access to private workspaces. The first administrator and email settings are configured during initial setup. New accounts receive a welcome email with application, release-note, and issue-report links.

Workspaces or selected pages can be exported as editable `.mindflow` files and imported into new or existing workspaces. PDF and PNG exports create one flattened output per canvas page.

## Product boundaries

- Browser targets are desktop Chrome/Firefox and iPad Safari, with finger and Pencil input. The tablet canvas supports one-finger tools and two-finger pan/zoom. Small-screen phone layouts are outside the current scope.
- One application instance serves a SQLite database on the operator's server; TrueNAS is a deployment option.
- Accounts, documents, original images, variants, and settings are stored on the server.
- Loaded pages can be edited temporarily while disconnected, but unsaved drafts live only in browser-session memory. This is not a persistent offline application.
- Sharing requires an existing account. A page URL is a destination, not a public-access grant.
- Collaboration uses one writer per page, with comments available independently of editor control.

See the [user guide](../user-docs/using-mindflow.md) for current behavior and the [roadmap](roadmap.md) for features that are not yet available.
