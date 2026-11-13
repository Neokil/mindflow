# Using mindflow

Sign in to see your own and shared workspaces. Create a workspace, open it, and use the page tabs to organize separate canvases. Workspaces and pages can be renamed, duplicated, or deleted; page tabs can be reordered. Workspace lifecycle and sharing management belong to the owner.

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

The top-right button opens **Canvas settings**: background, grid spacing, and snapping. The default is a dot grid with 20-unit spacing and snapping off. **Selection properties** in the bottom toolbar or right-clicking an object opens a separate panel beside that button or cursor. The panels are independent.

To crop, select one image and choose **Crop image** in its properties. Draw/adjust the crop rectangle or use percentage trims, then Apply. Cancel discards the draft; Reset crop followed by Apply restores the original. Cropping supports undo/redo and does not alter the original file.

Open **Your profile** beside your email to upload or remove an avatar. Avatars use PNG/JPEG/WebP up to 2 MiB, 8 megapixels, and 8,192 pixels per side, and are cropped to a square. Initials appear when no avatar is set. The appearance button cycles system/light/dark mode.

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
