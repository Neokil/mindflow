# Import, export, and migration

Use **Import workspace** on the overview to bring files into new private workspaces. Select several files to migrate a collection. Use **Import pages** in an open workspace to append supported pages. Existing pages are preserved. Imports never copy sharing permissions or grant access to another user's media.

Use **Export workspace**, or a page's **Export page** action, to choose a whole workspace or selected pages. Exports fit all content with a margin. Export saves pending edits first; a failed save stops the export.

## Formats and editability

| Format | Import | Export | What remains editable |
| --- | --- | --- | --- |
| mindflow `.mindflow` | New workspace or append | Workspace or selected pages | Native elements, pressure samples, groups/locks, optional templates, original media, and links whose targets are included. This is the most complete transfer format. |
| Butterfly `.bfly` | New workspace | Workspace or selected pages | Supported version-13 text, images, shapes, and strokes. Stars export as filled pen paths; typography, pressure, and highlighting can differ. |
| Goodnotes `.goodnotes` **experimental** | New workspace or append | Workspace or selected pages | Supported modern ZIP notebook text boxes, ink, and images. Exports contain native records, not a PDF renamed as `.goodnotes`. Destination-app compatibility still needs manual verification. |
| PDF `.pdf` | New workspace, append, or the editor's PDF placement options | Single or multi-page PDF | Imported page images can be annotated; exported canvas content is flattened. Each selected canvas becomes one PDF page. |
| PNG `.png` / `.zip` | Place images with the image tool | One image per selected canvas | Flattened images. Several pages download together in a ZIP. |

These are file conversions. They do not connect to cloud accounts or synchronize changes with other applications. Keep the original files and compare an imported/exported copy before replacing a notebook. Conversion notes report known omissions.

### Goodnotes details

Modern Goodnotes ZIP notebooks use an undocumented format. Import supports the observed protobuf records, Apple LZ4/TPL ink encodings, text boxes, image placement/crops, and embedded PDF backgrounds. Deleted objects are excluded when their records identify them as deleted. Rich text becomes one style per text box. Some newer ink encodings, shapes, stickers, recordings, links, and comments are unsupported and can be omitted with a conversion note.

Very large source pages split into continuation pages of at most 5,000 elements, retaining object coordinates and stacking order within each continuation. This keeps imports within canvas limits without silently dropping strokes. Embedded PDF pages become images. Fonts, pressure rendering, curves, and highlighter blending may differ.

On export, ordinary text and ink use native editable records. Shapes and rotated text become individual images; image rotation and crops remain represented separately. Pressure-varying strokes become constant-width editable strokes. Interactive list markers become visible text. A background PDF defines each exported page; very large canvases scale to fit PDF page limits. **Goodnotes itself is not part of automated CI:** the native export is experimental until checked in the destination app.

### Migrating from Microsoft OneNote

Export your OneNote pages or notebook to PDF, then import that PDF into mindflow. PDF import creates images that you can annotate; original text and ink are flattened. Native `.one` and `.onepkg` files are not supported. To move mindflow content into OneNote, export PDF or PNG images and insert them using OneNote's normal import/printout tools.

## Bulk import

1. Choose **Import workspace** and select up to 20 files, including different supported formats.
2. Start the import. Files run sequentially, each with its own result and conversion notes.
3. Open any successful workspace. A failed file does not undo other files.
4. Retry failed files without duplicating successful imports, or stop after the current file finishes.

Each non-PDF file is limited to 64 MiB; PDFs to 50 MiB. A batch is limited to 512 MiB. Native transfers allow 100 pages, 1,000 media assets, 128 MiB expanded, and a 32 MiB manifest. Individual canvases allow 10,000 objects and 20,000 samples per stroke. Goodnotes conversion runs in a worker with a two-minute timeout. Split larger notebooks into smaller source exports when necessary. The UI displays a useful error instead of partially saving a failed file.

## PDF migration from other applications

If an application has no compatible native format, export its notebook as a PDF and import that. The overview import creates one canvas per PDF page. The editor also offers page images on one canvas or separate canvases, plus an embedded PDF reader. Image imports support annotations on top; the embedded reader keeps the original PDF but cannot be annotated directly. Password-protected PDFs must be exported unlocked first.

For a workspace PDF export, choose **Whole workspace** and **PDF**. A canvas is infinite, so the export fits all elements instead of using a fixed paper size or the current viewport. Use selected-page exports to split very large documents.

## Screenshot export

Choose the camera button beside canvas settings. Drag a rectangle over the visible canvas, then **Download PNG** or **Copy to clipboard**. The rectangle uses the current viewport and zoom, exports only the chosen region, and does not modify objects or the current selection. Cancel or press Escape to return to editing. Touch, Pencil, and mouse use the same region overlay.

Clipboard image writing requires HTTPS (or localhost) and browser support. If it is unavailable, its button is disabled with an explanation; PNG downloads remain available. Clipboard permissions or browser policy may still reject a write. The screenshot includes the configured canvas background but excludes app chrome, sharing dialogs, comments, and selection handles.

See the [manual interoperability checklist](interoperability-testing.md) for destination-app checks and the [developer format notes](../developer-docs/interoperability.md) for implementation and validation boundaries.
