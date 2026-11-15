# Interoperability implementation

`workspace/transfer` obtains the same authorized, consistent native snapshot for mindflow and Butterfly exports. Imports ultimately use the existing atomic portable-file importer, which validates the complete document and assets before committing. Domain storage stays in `workspace/storage`; converters do not query tables or resolve arbitrary URLs.

## Native adapters

- **Butterfly:** `workspace/butterfly` reads and writes the supported version-13 JSON/ZIP format. See the [mapping](butterfly-import.md).
- **Goodnotes:** frontend `features/transfer/goodnotes` separates bounded ZIP/protobuf/LZ4/TPL decoding, ink, native writing, and UI conversion. CPU-heavy conversion runs in a terminable worker. PDF.js renders embedded PDF pages on the browser side, then the normal validated `.mindflow` importer commits the result. Exports start from the server's permission-checked native archive. The writer creates native notebook/index/page/attachment records; unsupported shapes/rotated text become individual images through the shared canvas renderer. Format research is attributed in `THIRD_PARTY_NOTICES.md`. This undocumented format is experimental; parser round trips are not proof that Goodnotes accepts an export.

All converters are local. They do not send notebooks to external services, execute file-chosen commands, or extract archive paths into the filesystem. File names choose only allowlisted formats. Compressed/expanded budgets, document/sample limits, and timeouts apply independently. Generated raster media is budgeted incrementally before retention, and the Go conversion deadline also covers image preparation. Goodnotes export reuses one decoded archive rather than duplicating media buffers. The private real-world samples used during development remain outside Git.

## Build and dependency maintenance

The Go server and TypeScript frontend are built with the normal project toolchain. The final Docker image remains `scratch`. There is no external notebook codec or .NET toolchain. Use PDF as the documented OneNote migration path.

## Browser transfer flow

Bulk import runs sequentially with a result per file, a stop-after-current flag, and retry that skips successful files. Client-side PDF/Goodnotes conversion finishes before the atomic server import. A failed conversion creates no workspace. Butterfly currently creates new workspaces only.

Screenshot export uses a pointer overlay over the visible canvas. Screen coordinates become world coordinates through the current camera; `renderPage` accepts an exact region rather than fitted content bounds. Pending saves complete first. Clipboard writes begin synchronously from the user's click with a promised PNG payload for browsers requiring user activation; download is the fallback. Neither document content nor editor selection is modified.

## Verification boundaries

Go integration tests cover authorization and Butterfly round trips. Frontend tests exercise binary framing, expansion rejection, and native Goodnotes round trips. Playwright covers native import/export, bulk error/retry behavior, PDF placement/exports, and screenshot PNG dimensions/clipboard.

The private Goodnotes sample is an opt-in local check. Goodnotes browser coverage accepts `MINDFLOW_GOODNOTES_FIXTURE=/path/to/file.goodnotes`; it is never required by CI. Test fixtures under `docs/examples` and the small unit-test ZIP are synthetic. Destination app editing/visual fidelity requires the [manual checklist](../user-docs/interoperability-testing.md); Destination apps are not available on CI runners.
