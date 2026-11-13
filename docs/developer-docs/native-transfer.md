# Native import and export

The workspace `transfer` subdomain creates portable `.mindflow` archives and imports them into a new private workspace or appends their pages to an editable workspace. SQL and generated bindings stay in the parent `workspace/storage` package. There are no new tables or migrations.

## Version 1 file format

A `.mindflow` file is a ZIP with these exact entries:

- `manifest.json`: UTF-8 JSON manifest.
- `assets/{id}.bin`: original bytes of each referenced PNG, JPEG, WebP, or retained PDF.
- `assets/{id}.png`: required first-page PNG preview for each retained PDF.

No paths are extracted. Portable asset keys use 1–80 ASCII letters, digits, underscores or hyphens. Other files, symlinks, encrypted entries, duplicates, missing references, and unsupported versions are rejected.

```json
{
  "format": "mindflow",
  "version": 1,
  "kind": "workspace",
  "name": "Ideas",
  "pages": [
    {
      "name": "Page 1",
      "document": {
        "schemaVersion": 1,
        "elements": [],
        "settings": { "background": "dots", "gridSize": 20, "snap": false }
      }
    }
  ],
  "templates": [],
  "assets": []
}
```

`kind` is `workspace` for a complete workspace and `pages` for a selection. Array order defines page order. Each template contains `name` and `elements`. Assets contain `id`, `mime`, and `preview: true` for PDFs; element `assetId` values reference these portable keys. Canvas elements/settings use the normal schema. Only referenced originals are included; display variants are regenerated as needed.

Permissions, accounts, comments, editor leases, viewport positions, revisions, and undo history are excluded. Native files contain unencrypted original media, including portions hidden by image crops, so share them accordingly.

## HTTP and transactions

All routes require authentication and CSRF protection:

| Route | Result |
| --- | --- |
| `POST /workspaces/{id}/export` | Binary ZIP attachment (`application/vnd.mindflow+zip`). |
| `POST /workspaces/{id}/export/snapshot` | Authorized saved pages for browser PDF/PNG rendering. |
| `POST /workspaces/import/mindflow` | Multipart `file`, optional `name`; new private workspace. |
| `POST /workspaces/{id}/import/mindflow` | Multipart `file`; append pages and optional templates. |

Export/snapshot bodies contain `pageIds` (empty means all, otherwise validated against this workspace) and `includeTemplates`. Selected pages always follow workspace order. Templates are optional, independently of the page selection.

Every current reader can export. Appending requires owner/full access, checked again inside the write transaction. Administrators have no implicit access. A read transaction captures each export consistently. Imports validate the complete archive and decode media before writing, then use one transaction for the workspace, pages, assets, and templates. Failure rolls everything back. Content receives fresh IDs. Existing pages are untouched; colliding template names gain an `(imported N)` suffix. A single transfer operation runs at a time per server process.

## Limits

| Resource | Limit |
| --- | --- |
| Native compressed archive | 64 MiB |
| Expanded archive, including manifest | 128 MiB |
| Manifest | 32 MiB |
| Pages | 100 per transfer; complete empty workspaces allowed |
| Page document | 16 MiB; normal 10,000-element limit |
| Elements across pages/templates | 100,000 |
| Assets | 1,000 |
| Templates | 100; 500 elements and 1 MiB per template |
| Media | Existing image/PDF upload limits apply |

Counts and actual decompressed bytes are bounded; ZIP metadata alone is not trusted. Native imports do not resolve URLs or execute embedded content.

## PDF and PNG rendering

`web/src/features/transfer` separates dialog controls, bounds calculations, rendering, and output packaging. PDF/ZIP libraries load on demand. Rendering uses the shared `CanvasElement` content component in a temporary Konva stage, preloading media and fonts before drawing. Export does not include selection handles, collaboration overlays, or the PDF reader caption.

Bounds include all elements, rotation, border/stroke thickness and a 32-unit margin. Empty pages use a 1,000 × 700-unit rectangle. The camera does not influence exports. Each output uses the selected 1,024/2,048/4,096-pixel longest edge. Workspace PDFs contain one raster page per canvas; PNG exports produce one PNG, or an ordered ZIP when multiple pages are selected. PDF page proportions follow canvas bounds, with a maximum 14,400-point edge. These are visual exports, not editable backups or selectable vector-text PDFs. Retained PDFs render their first-page preview. Comments are omitted.

Background and dot-grid inclusion are independent options; the current appearance supplies the fallback background color. Grids finer than four output pixels are omitted. Total output pixels are limited to 128 megapixels and rendered PNG bytes to 128 MiB. Processing is sequential between pages. Keep the tab open until completion. Current-tab drafts must save successfully before an export starts; offline/recovery drafts block export rather than silently producing stale content.
