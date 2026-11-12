# mindflow — Product requirements

Status: agreed product scope for the first release, based on the requirements interview of 12 November 2026. Technical architecture and implementation choices will be decided separately.

## Product goal

mindflow is a minimalistic, self-hosted web application for arranging images, shapes, text, and freehand drawings on infinite canvases. Its interface should feel at home alongside current Apple applications: restrained, clear, and focused on the canvas.

The first release serves the owner and invited users on desktop browsers. Each user's content is private. The application is intended to run on the owner's server, preferably as an app on TrueNAS; deployment details remain undecided.

## First-release scope

- Invite-only accounts with email and password authentication.
- Initial setup for the first administrator and SMTP email delivery.
- Private workspaces containing multiple pages displayed as tabs.
- Infinite canvases with panning, cursor-centered zoom, and freehand lasso selection.
- Images, geometric shapes, basic formatted text, and mouse-drawn pen strokes.
- Element manipulation, grouping, stacking, and optional snap-to-grid.
- Automatic server saving and temporary disconnected editing within the open session.
- Light and dark appearance with configurable canvas backgrounds.
- Account administration, invitations, and password recovery.

## Accounts, setup, and privacy

### Initial setup

On first launch, a setup screen lets the owner create the first administrator and configure SMTP for invitation and password-reset emails. After initialization, the first-run setup must no longer allow visitors to create an administrator.

There is no public self-registration.

### Invitations and authentication

1. An administrator creates an invitation using the recipient's email address.
2. The recipient receives an email with an invitation link.
3. The link allows the recipient to set up their account and password.
4. Once activated, the user can sign in with email and password.

Administrators can resend invitations. Users have a “Forgot password” flow that sends a password-reset email. Administrators can also trigger a password-reset email for a user; they do not set or view that user's password through this flow.

Invitation and password-reset links must be time-limited and single-use. Exact expiration periods will be decided during technical design. Expired or invalid links must produce a clear message and a path to request help or a replacement.

Users can sign out. Unauthenticated visitors cannot access workspaces, pages, images, or other user content.

### Administrator capabilities

An administrator is a normal user with access to an additional user-management screen. Administrators can:

- Create users through email invitations.
- Resend invitations.
- Deactivate users.
- Delete users.
- Send password-reset emails.

Deactivation prevents access without deleting the user's content. It must also prevent continued access through an existing session.

Deleting a user requires explicit confirmation and permanently deletes their workspaces, pages, images, and all related content.

### Content isolation

Every workspace, page, and associated asset belongs to its owning user. Users can only access their own content. Administrator status does not grant access to another user's workspaces, files, or canvas content, including through direct links or application requests.

This is an application access rule. Infrastructure operators' access to server storage is outside the scope of the product permission model.

## Workspaces and pages

After signing in, a user sees a list of the workspaces they created.

Users can create, open, rename, duplicate, and delete workspaces. Opening a workspace displays its pages as tabs at the top of the editor. Users can create, rename, duplicate, delete, switch between, and reorder pages.

Each page has its own canvas. Switching tabs must not discard pending changes. Duplicating a page copies its content into an independently editable page. Duplicating a workspace copies its pages and content into an independently editable workspace. Editing or deleting an original must not break its duplicate.

Destructive workspace and page deletion should require confirmation.

## Canvas navigation and selection

The canvas has no user-facing page boundary and can be panned in every direction. There is no fixed document size.

| Input | Behavior |
| --- | --- |
| Scroll wheel or trackpad scroll | Zoom around the cursor position. |
| Cmd + drag on macOS | Pan the canvas. |
| Ctrl + drag on Windows/Linux | Pan the canvas. |
| Click and drag empty canvas in selection mode | Draw a freehand lasso to select elements. |
| Click an element in selection mode | Select the element. |
| Drag a selected element | Move the selection. |

Drawing and placement tools take precedence over lasso selection while active. The interface must make the active tool clear. Lasso selection is a freehand outline, not a rectangular selection box.

Multiple elements can be selected and manipulated together. Exact lasso intersection rules, keyboard shortcuts beyond the navigation controls above, and zoom limits remain interaction-design details.

## Canvas elements

### Shared editing behavior

All element types support selection, movement, resizing, rotation, duplication, copy/paste, and deletion. Users can also:

- Select multiple elements.
- Group and ungroup elements for manipulation together.
- Bring elements forward and send them backward in the stacking order.
- Undo and redo edits within the current editing session.
- Hovering a movable element shows a grab cursor; dragging shows a grabbing cursor. Drawing tools and read-only elements keep their appropriate cursors.
- Turn snap-to-grid on and off.

Snap-to-grid is a positioning aid. Its toggle is separate from the visible background choice, so showing dots need not force snapping. Exact grid spacing and snapping rules will be decided during interaction design.

Element locking is outside the first-release scope.

### Images

Images can be added through file upload, drag-and-drop, and clipboard paste. Each insertion method enters placement mode: the pending image follows the cursor, and a click places it on the canvas. The user must also be able to cancel placement.

- Corner resize handles preserve the image's aspect ratio.
- Side resize handles resize without preserving aspect ratio, allowing stretching.
- Images support rotation and the shared editing operations.
- Images are stored with the user's server-side content.
- Images can be cropped using a preview with a movable rectangle, corner handles and percentage controls. Apply commits the crop; Cancel leaves the image unchanged. Reset restores the full original. Cropping preserves the original file and supports undo/redo.

Adding images by URL is outside the first-release scope. Supported image formats and upload limits will be determined during technical design.

### Shapes

Initial shape types are circle, rectangle, and star. The product should allow additional shape types in later releases.

Shapes support configurable fill color, border color, and border thickness, along with the shared editing operations. Shapes do not contain editable text in the first release; separate text elements can be placed nearby or over them.

### Text

Users can create and edit text elements with basic formatting:

- Font size in whole numbers (1–2,000); inputs reject fractions and corner resizing rounds to a whole number.
- Bold and italic.
- Horizontal text alignment (left, center, right).
- Vertical text alignment (top, middle, bottom), including while editing.
- Text color.

Text elements support the shared editing operations, including rotation and resizing. The precise effect of resizing on text wrapping and font size will be settled during interaction design.

Rich text features such as lists, headings, and link editing are outside the first-release scope.

### Freehand drawing

A pen tool allows users to draw with the mouse, using adjustable color and thickness. Each completed stroke becomes a selectable element that can be moved, resized, rotated, and deleted like other elements.

There is no eraser in the first release. Users remove unwanted strokes by selecting and deleting them.

## Saving and disconnected editing

Workspaces, pages, canvas elements, and uploaded images are stored on the server. Changes save automatically without requiring a manual save action.

The interface must clearly distinguish saving, saved, disconnected, and save-failure states. It must never imply that unsaved changes are stored on the server.

When connectivity is lost:

1. Display a clear warning.
2. Allow continued editing within the open session using the content already available there.
3. Keep pending changes in session memory.
4. Retry saving automatically when connectivity returns.
5. Show a saved state only after pending changes have been accepted by the server.

Unsaved changes do not need to survive closing the tab, reloading, or browser failure. If a user attempts to leave, reload, or close a session with unsaved changes, request the browser's unsaved-changes confirmation so the user can cancel leaving. Browser-controlled wording and platform restrictions apply; a custom dialog cannot be guaranteed for closing a tab or browser.

Undo/redo history is limited to the current editing session. Persisted version history and full offline operation are outside the first-release scope. Editing the same page in multiple sessions requires a defined conflict policy during technical design so that pending changes cannot be silently overwritten.

## Visual and interaction design

- Minimalistic interface inspired by current Apple applications.
- A clear hierarchy, restrained controls, and generous usable canvas space.
- Page tabs at the top of an open workspace.
- Light and dark modes, following the system appearance by default.
- Configurable canvas background, with a subtle dot grid as the default and a plain background option.
- Discoverable tools and clear selection, placement, connection, and saving states.
- Desktop support for macOS, Windows, and Linux browsers; the exact browser support matrix will be decided later.

Mobile editing, iPad-specific interactions, Apple Pencil support, and pressure-sensitive drawing are outside the first-release scope.

## Deployment and operational requirements

- The app runs on the owner's server, with TrueNAS as the preferred hosting environment.
- SMTP settings are configured during initial setup.
- Server-side data must persist across application restarts and upgrades.
- Credentials and private content must be protected in deployment; technical mechanisms will be specified separately.
- Server backup and restore procedures will be defined with deployment design. In-app export is not required for the first release.

No frontend framework, backend framework, database, storage layout, packaging approach, or TrueNAS version is selected by this document.

## Acceptance criteria

The first release is ready when the following user journeys work:

1. **Initialize:** A fresh installation allows the owner to create an administrator and configure SMTP. Subsequent visitors cannot repeat administrator setup.
2. **Invite:** An administrator invites a user by email. The recipient sets a password through the invitation link and can sign in. Unknown visitors cannot self-register.
3. **Recover access:** A user requests a password-reset email and resets their password. An administrator can resend an invite or trigger a reset email.
4. **Manage accounts:** An administrator can deactivate and delete accounts. Deactivation blocks existing sessions; confirmed deletion removes all content belonging to the deleted user.
5. **Keep content private:** A second user, including an administrator, cannot access the first user's content through the UI, direct asset links, or application requests.
6. **Organize work:** A user can create, rename, duplicate, and delete workspaces and pages, and reorder page tabs. Copies remain independent of their originals.
7. **Navigate:** A user can pan in every direction, zoom around the cursor using scroll input, and select elements using a freehand lasso.
8. **Compose:** A user can insert images through all three supported methods, position them with the cursor, add each initial shape type, format text, and draw pen strokes.
9. **Transform:** All element types can be moved, resized, and rotated. Image corners preserve aspect ratio; side handles allow stretching. Multi-selection, grouping, stacking, duplication, copy/paste, deletion, undo/redo, and optional snapping work.
10. **Save:** After edits are saved, reopening a workspace restores its content and images from the server.
11. **Reconnect:** A user can continue editing during a connection loss. Reconnecting saves pending changes, and attempting to leave with unsaved changes requests a browser confirmation.
12. **Appearance:** The app supports light and dark modes and configurable canvas backgrounds, defaulting to dots.

## Future roadmap

These features are explicitly deferred:

- Shared workspaces and collaboration; simultaneous editing is not included initially.
- Export, with formats to be decided later.
- Import from Linwood Butterfly (`butterfly.linwood.dev`); compatibility and import fidelity require future investigation.
- Additional shape types.

## Decisions for later design

- Technical stack, storage, deployment packaging, and TrueNAS compatibility.
- Browser support matrix, performance targets, image formats, and upload limits.
- SMTP configuration maintenance, invitation expiration, and account security details.
- Protection against deleting or deactivating the last administrator.
- Lasso inclusion rules, detailed keyboard shortcuts, grid spacing, and text resize behavior.
- Handling concurrent sessions, retries, and conflicting saves.
- Backup and restore procedures.

These details must be resolved as needed before implementation; they do not expand the agreed first-release feature scope.

## Technical interview clarifications — 12 November 2026

The following product behavior was clarified during technical design. These decisions refine the sections above:

- Browser support targets current stable desktop Chrome and Firefox. Safari and Edge are not part of the initial acceptance matrix.
- One session can edit a page at a time. Other sessions open it read-only, see saved updates automatically, and can explicitly take control, making the prior editor read-only.
- If takeover displaces unsaved changes, retain them in memory and offer saving them as a separate page rather than overwriting newer work.
- Text formatting applies to the whole element. Side handles change width/wrapping; corner handles scale the text and font size.
- The freehand lasso selects only elements it fully contains.
- Grid spacing defaults to 20 canvas units, is configurable, and applies when moving and resizing with snapping enabled.
- Accept PNG, JPEG, and WebP images, initially limited to 20 MB each. Keep originals and generate smaller variants when useful, with a reduction threshold of more than 20%, accounting for zoom and display density.
- Invitations expire after seven days, password-reset links after one hour, and login sessions after 30 days unless revoked earlier.
- Prevent deletion or deactivation of the last active administrator.
- Administrators can update SMTP settings and the public application URL after setup and send a test email.
- TrueNAS manages backups. All durable application data, including images, is stored in SQLite.

See [ARCHITECTURE.md](ARCHITECTURE.md), [CANVAS_AND_SYNC.md](CANVAS_AND_SYNC.md), and [DEPLOYMENT.md](DEPLOYMENT.md) for the technical drafts. Proposed implementation defaults in those documents are distinguished from interview decisions.
