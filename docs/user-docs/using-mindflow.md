# Using mindflow

Sign in to see your own and shared workspaces. Create a workspace, open it, and use the page tabs to organize separate canvases. Workspaces and pages can be renamed, duplicated, or deleted; page tabs can be reordered. Workspace lifecycle and sharing management belong to the owner.

## Import from Butterfly

On the workspace list, choose **Import workspace**, select a `.bfly` export, enter a workspace name, and choose **Import**. Review the conversion notes, then choose **Open workspace**. The new workspace is private; existing workspaces and the source file are unchanged. A failed import leaves no partial workspace.

The importer supports unencrypted Butterfly **file format version 13** documents with numbered pages. Page names, order, stacking, positions, sizes, rotations, collection groups, and solid background colors are preserved. Supported content becomes editable mindflow elements:

- Embedded PNG/JPEG/WebP images retain their originals and display dimensions.
- Pen paths become selectable, resizable strokes. Pressure, smoothing, and filled strokes are approximated.
- Rectangle and circle/ellipse shapes preserve solid outlines and fills.
- Text preserves its content, one style per element, color, whole-number font size, and basic alignment. Empty labels are omitted. Fonts, rich text, and advanced layout may differ.

Exports can be up to **64 MiB**, with at most **100 pages**. Individual images follow the normal upload limits. External image URLs are never fetched. Other format versions, encrypted files, nested page folders, corrupt or oversized data, and unsupported image constraints are rejected. Unsupported element types (including PDF/SVG elements), gradient shapes, layered/textured backgrounds, areas, waypoints, and animations are skipped or simplified with conversion notes. Keep the original export if you need those features.

Pages open fitted to their content; your camera position is retained when switching pages during the session. Imported background colors are configurable in **Canvas settings**.

## Canvas controls

| Action | Control |
| --- | --- |
| Zoom around the cursor | Scroll. |
| Pan | Cmd/Ctrl + drag, or hold the middle mouse button and drag. |
| Select enclosed elements | Drag a freehand lasso in Select mode. An element must be fully contained. |
| Add to selection | Shift + click or lasso. |
| Create an element | Choose a tool, drag to preview its size, and release. Shift constrains it to 1:1; a click uses its default size. |
| Undo / redo | Cmd/Ctrl + Z / Shift + Z, or the toolbar. History is kept only in the current session. |
| Copy / paste / duplicate | Cmd/Ctrl + C / V / D. |
| Group / ungroup | Cmd/Ctrl + G / Shift + G. |
| Delete | Delete/Backspace. Pen strokes are selectable elements; there is no eraser. |
| Finish editing text | Cmd/Ctrl + Enter. |
| Cancel | Escape. |

The keyboard button in the editor lists shortcuts. Hovering over movable elements shows a grab cursor; dragging shows a grabbing cursor.

## Elements and settings

Add images by upload, drag-and-drop, or clipboard paste. Once ready, an image follows the cursor until placed; drag to size it or click for its default size. PNG, JPEG, and WebP uploads support up to 20 MiB, 40 megapixels, and 16,384 pixels per side. Unsupported animated images are rejected. Originals are retained and smaller display variants are generated when useful.

All element types support moving, resizing, and rotation. Image corner handles preserve aspect ratio; side handles stretch. Text starts centered horizontally and vertically, with whole-number font sizes (1–2,000), bold/italic, color, and alignment settings. The pen has adjustable color and thickness.

The top-right button opens **Canvas settings**: background, an optional custom background color, grid spacing, and snapping. The default is a dot grid with 20-unit spacing and snapping off. **Selection properties** in the bottom toolbar or right-clicking an object opens a separate panel beside that button or cursor. The panels are independent.

To crop, select one image and choose **Crop image** in its properties. Draw/adjust the crop rectangle or use percentage trims, then Apply. Cancel discards the draft; Reset crop followed by Apply restores the original. Cropping supports undo/redo and does not alter the original file.

Open **Your profile** beside your email to upload or remove an avatar. Avatars use PNG/JPEG/WebP up to 2 MiB, 8 megapixels, and 8,192 pixels per side, and are cropped to a square. Initials appear when no avatar is set. Appearance follows your system preference by default and updates when it changes. The appearance button cycles system/light/dark mode if you want an explicit override.

## Workspace templates and locked groups

Select one or more elements and choose **Save as template** in the bottom selection bar or object properties. Give it a name such as “Heading”. Choose **Workspace templates** in the drawing toolbar on any page, select **Insert**, and click on the canvas to place the preview. Escape cancels placement.

Templates are snapshots of the selection, including images and relative placement. Insertions get fresh identities and start ungrouped and unlocked, so each element can be edited independently. Editing an original, template copy, or another instance does not change the others. Original images remain available even if the source page is deleted. The library supports up to 100 templates per workspace, 500 elements and 1 MiB of element data per template. Names must be unique within the workspace.

All workspace members can view the library. Owners and full-access members can save or delete templates; inserting requires page editor control. Deleting a template leaves inserted elements unchanged. Duplicating a workspace also copies its template library into the new private workspace.

To fix objects in place, select them, group them with Cmd/Ctrl + G, and choose **Lock groups** in the selection bar or object properties. Locked groups remain selectable and show dashed outlines, but cannot be dragged, resized, rotated, edited, reordered, ungrouped, or deleted through the editor. A selection containing a locked group is protected as a whole. Choose **Unlock groups** before changing it. Locking is saved with the page and supports undo/redo; an editor with full access and control can unlock a group. It is an editing safeguard, not a separate access permission. Copies and templates can still be made from locked content.

## Sharing and comments

Click **Private** beside the workspace name to add an existing active user's email. The badge becomes **Shared**; every member can see who owns the workspace.

| Access | Capabilities |
| --- | --- |
| Read-only | View saved pages, images, and comments. |
| Comment | Read and add comment threads/replies. |
| Full access | Comment and edit page content, manage pages, and request editor control. |
| Owner | Full access plus sharing, workspace lifecycle, and force control. |

Only the owner changes membership. Sharing does not send account invitations or create a public-access link. Removing the last member makes the workspace private again. Duplicated workspaces start private. Workspace/page copies do not copy comment threads; page copies in a shared workspace remain accessible through that workspace.

Use **Add comment** and click a canvas point. Hover or focus the avatar/initials pin to read the thread and reply. The creator or workspace owner can resolve, reopen, or delete a thread. **Show resolved comments** reveals resolved pins. Commenting does not require the canvas editor lease.

## Editor control and recovery

Each page has one writer. A second session sees saved content read-only and can choose **Take control**. The current writer can accept or deny; after ten seconds the request is automatically accepted. The requester can cancel. The workspace owner can choose **Force control** to transfer immediately. Administrators who are not owners have no special control privilege.

Autosave sends completed edits to the server. Check the save status before leaving. During a connection loss, loaded content can still be edited in memory. If another session takes control, your unsaved changes are kept separately and can be saved as a recovered page. Keep that tab open until recovery is saved. Browser crashes or forced closure can lose drafts; there is no persistent local offline store.

## Links and application updates

Copy the current workspace/page URL to send an authorized user directly to that page. It keeps its destination through sign-in but grants no access. Browser Back/Forward navigates pages and workspaces; leaving with unsaved work asks for confirmation.

After a new server build is deployed, existing tabs show a reload banner. Reload is explicit and warns about unsaved work. Tabs predating this notification feature need one initial reload to install it.
