# Product definition

mindflow provides a quiet, flexible space for collecting images, arranging thoughts, sketching, and discussing ideas. It is a self-hosted web application for individuals and small groups of invited users, with desktop and tablet support and an interface inspired by Apple applications.

## What it does

Users organize their work in private-by-default workspaces containing tabbed infinite-canvas pages. Pages support images, rectangle/circle/star shapes, text with element-wide formatting, lists and checkable checklists, highlighter marks, retained PDFs, and freehand pen strokes with optional Pencil pressure, adjustable stabilization, and hold-to-fit lines, circles, and rectangles. Elements can be selected, moved, resized, rotated, grouped, duplicated, cropped where applicable, aligned, linked to other elements on any page of the workspace, and arranged in layers.

Butterfly version-13 exports can be imported into new private workspaces, preserving ordered pages, embedded images, text, freehand strokes, and rectangle/ellipse shapes. Conversion notes identify approximations and unsupported content.

Selections can be saved as workspace templates and inserted on other pages as independently editable copies. Groups can be locked to protect them against accidental editing or movement.

Workspaces can be shared with existing users as read-only, comment, or full access. Comment threads attach conversations to canvas points. Each page has one writer at a time; other users can request control, and the workspace owner can force a transfer. Saved changes are visible to other sessions. Owners can transfer a workspace to another active user and retain full access. Deleted workspaces/pages can be restored from Trash for 30 days before permanent removal.

Accounts are invite-only. Administrators manage users and invitations without gaining access to private workspaces. The first administrator and email settings are configured during initial setup. New accounts receive a welcome email with application, user-guide, release-note, and issue-report links. Users can opt into upgrade emails and unsubscribe with one click. Active non-admin accounts follow a delivery-aware inactivity reminder and deletion policy. Private in-app bug reports and feature requests support image attachments, discussion, unread badges, administrator search, and a created/accepted/closed lifecycle with release-linked bug closure. Closed reports expire after 30 days unless reopened.

Workspaces or selected pages can be exported as editable `.mindflow` files and imported into new or existing workspaces. PDF exports combine selected canvases into one document with a flattened page per canvas; PNG exports produce one image per canvas, packaged as a ZIP when there are multiple images.

## Product boundaries

- Browser targets are desktop Chrome/Firefox and iPad Safari, with finger and Pencil input. The tablet canvas supports one-finger tools and two-finger pan/zoom. Small-screen phone layouts are outside the current scope.
- One application instance serves a SQLite database on the operator's server; TrueNAS is a deployment option.
- Accounts, documents, original images, variants, and settings are stored on the server.
- Loaded pages can be edited temporarily while disconnected, but unsaved drafts live only in browser-session memory. This is not a persistent offline application.
- Sharing requires an existing account. A page URL is a destination, not a public-access grant.
- Collaboration uses one writer per page, with comments available independently of editor control.

See the [user guide](../user-docs/using-mindflow.md) for current behavior and the [roadmap](roadmap.md) for features that are not yet available.
