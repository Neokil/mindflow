# Butterfly import and export

`internal/workspace/butterfly` is a workspace subdomain with explicit workspace, page, and image collaborators. It owns archive parsing and conversion; all SQL and schema ownership remain in `workspace/storage`. No database migration or runtime dependency is added.

## Request and transaction

Authenticated `POST /api/v1/workspaces/import/butterfly` accepts multipart `file` and optional `name`. Normal origin, CSRF, and session middleware apply. The response contains `id`, first `pageId`, `pageCount`, `elementCount`, and deduplicated `warnings`.

The service validates the archive and prepares supported images before opening a transaction. One parent-storage transaction creates a private workspace, scopes its original assets to the owner, creates every converted page, and retains page/asset associations. All writes roll back on failure. Notifications publish after commit. Imports are serialized with a nonblocking service mutex to bound concurrent conversion memory.

## Supported mapping

| Butterfly version-13 data | mindflow representation |
| --- | --- |
| `META-INF/butterfly.json` document metadata | Optional fallback workspace name; validates document type/version. |
| `BFLY/pages/<index>.<name>.json` | Pages sorted by numeric index, preserving names. |
| Layer content, back to front | Flat element stacking; collections become fresh group IDs per page. |
| Embedded image | Original asset plus editable placement with scaled/fixed dimensions. |
| Pen points | Normalized editable stroke; pressure/smoothing/fill approximated. |
| Solid rectangle/circle | Rectangle/ellipse with outline and fill. |
| Text spans | Plain text with one style and basic alignment; conservative bounds and mindflow font metrics. |
| First solid texture background color | Optional `settings.backgroundColor`, with plain background pattern. |

ARGB colors become validated CSS colors; degrees become radians; top-left coordinates become element centers. Unsupported types are skipped with notes. Unsupported gradients, shear, advanced backgrounds, text layout, and other approximations produce notes. Missing required data or unsupported image constraints abort the import. The [user guide](../user-docs/using-mindflow.md#import-from-butterfly) describes compatibility limits.

## Bounds and security

No archive entry is extracted to disk and no external URL is fetched. ZIP paths must be canonical and relative; duplicate paths, symbolic links, and encrypted entries are rejected. Declared and streamed sizes are bounded, and ZIP integrity errors fail the import.

| Limit | Value |
| --- | --- |
| ZIP / full multipart request | 64 / 65 MiB |
| Expanded archive | 128 MiB |
| Archive entries / pages | 2,048 / 100 |
| Page JSON / embedded image / metadata | 16 MiB / 20 MiB / 64 KiB |
| Elements per page / import | 10,000 / 50,000 |
| Points per stroke / text bytes per element | 20,000 / 100,000 |

Images pass through the existing image decoder, dimension, animation, and pixel limits. Imported assets use the normal private-workspace authorization and save validation. The source archive itself is not retained.

## Verification and format references

Parser tests cover conversion, ordering, unsafe paths, version checks, missing images, external references, and expansion bounds. API tests cover CSRF, private access, editable saves, and transactional rollback. `web/e2e/butterfly.spec.ts` imports through the dashboard in Chrome and Firefox and checks visible colors, image delivery, and subsequent editing. Its fixture is synthetic and contains no personal sample data.

To verify another equivalent two-page sample manually, set `MINDFLOW_IMPORT_SAMPLE` to its absolute path when running that Playwright spec. Normal CI uses the committed synthetic fixture.

Format mapping follows the upstream [Butterfly repository](https://github.com/LinwoodDev/Butterfly), especially the API models for [elements](https://github.com/LinwoodDev/Butterfly/blob/develop/api/lib/src/models/element.dart), [pages](https://github.com/LinwoodDev/Butterfly/blob/develop/api/lib/src/models/page.dart), and [text](https://github.com/LinwoodDev/Butterfly/blob/develop/api/lib/src/models/text.dart). This is a bounded converter, not a complete Butterfly renderer.

## Export

The transfer service first obtains its normal authorized, consistent native snapshot, then passes its validated page documents and embedded media to `butterfly.Write`. The writer emits file-format version 13, ordered page entries, a minimal tool palette, and embedded images. It defines no storage or cross-domain queries. The HTTP export selection accepts `format: "butterfly"`; conversion notes accompany the ZIP in `X-Mindflow-Conversion-Notes`.

Rectangles, circles, images, text, and constant-width strokes map to editable Butterfly elements. Stars become closed, filled pen paths. List markers become text; text metrics, pressure width, and multiply-blended highlighting can differ. Image crops are baked into PNG media; embedded PDFs use their retained first-page preview. Repeated identical media/crops share an archive entry. Sharing, comments, templates, links, locks, and mindflow-specific settings are omitted. Export bounds remain 64 MiB compressed and 128 MiB expanded. Round-trip tests cover API authorization, page/media counts, and reimport through the normal Butterfly parser.
