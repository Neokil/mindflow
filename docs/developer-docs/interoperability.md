# Interoperability implementation

`workspace/transfer` obtains the same authorized, consistent native snapshot for mindflow, Butterfly, and OneNote exports. Imports ultimately use the existing atomic portable-file importer, which validates the complete document and assets before committing. Domain storage stays in `workspace/storage`; converters do not query tables or resolve arbitrary URLs.

## Native adapters

- **Butterfly:** `workspace/butterfly` reads and writes the supported version-13 JSON/ZIP format. See the [mapping](butterfly-import.md).
- **OneNote:** `workspace/onenote` invokes the explicit `mindflow-onenote` executable. Its C# source uses OfficeIMO.OneNote/Core with a locked dependency graph. It compiles with .NET NativeAOT; production has no .NET CLI, interpreter, or SDK. The Go adapter allowlists four modes, sends file bytes through stdin, bounds stdout/stderr, serializes conversions, and applies a one-minute deadline. The helper has no database integration or HTTP client. AOT does not itself sandbox the process: it runs with the server's restricted container identity. Go prepares per-object PNG fallbacks for unsupported native representations while preserving normal text/ink as editable records.
- **Goodnotes:** frontend `features/transfer/goodnotes` separates bounded ZIP/protobuf/LZ4/TPL decoding, ink, native writing, and UI conversion. CPU-heavy conversion runs in a terminable worker. PDF.js renders embedded PDF pages on the browser side, then the normal validated `.mindflow` importer commits the result. Exports start from the server's permission-checked native archive. The writer creates native notebook/index/page/attachment records; unsupported shapes/rotated text become individual images through the shared canvas renderer. Format research is attributed in `THIRD_PARTY_NOTICES.md`. This undocumented format is experimental; parser round trips are not proof that Goodnotes accepts an export.

All converters are local. They do not send notebooks to external services, execute file-chosen commands, or extract archive paths into the filesystem. File names choose only allowlisted formats. Compressed/expanded budgets, document/sample limits, and timeouts apply independently. Generated raster media is budgeted incrementally before retention, and the Go conversion deadline also covers image preparation. Goodnotes export reuses one decoded archive rather than duplicating media buffers. The private real-world samples used during development remain outside Git.

## Build and dependency maintenance

Install .NET SDK 10 (CI pins the patched SDK), plus the platform's NativeAOT toolchain: Xcode command-line tools on macOS; clang, build tools, and zlib development headers on Linux. `make native-codec` publishes the host executable beside the browser-test server under `.cache/e2e`. `make dev` sets its path explicitly; `make build` copies it alongside the Go binary. `MINDFLOW_ONENOTE_CODEC` may override the trusted executable path. Otherwise the server looks beside its executable. Distribute both binaries together.

The Docker build downloads Microsoft's SDK with a verified SHA-512, pins the AOT runtime/compiler, and statically links musl/OpenSSL. The final image remains `scratch`, with a non-root user and both executables. Codec operation is tested in the read-only production container. `packages.lock.json` covers host and Linux amd64/arm64 runtime IDs; restore uses locked mode. `make upgrade` updates the SDK/checksums, runtime/compiler, OfficeIMO pin, workflow action references, and lock file. `make audit` includes NuGet advisories. Review bundled notices after updates, including static native libraries.

## Browser transfer flow

Bulk import runs sequentially with a result per file, a stop-after-current flag, and retry that skips successful files. Client-side PDF/Goodnotes conversion finishes before the atomic server import. A failed conversion creates no workspace. OneNote conversion happens server-side before the same import transaction. Butterfly currently creates new workspaces only.

Screenshot export uses a pointer overlay over the visible canvas. Screen coordinates become world coordinates through the current camera; `renderPage` accepts an exact region rather than fitted content bounds. Pending saves complete first. Clipboard writes begin synchronously from the user's click with a promised PNG payload for browsers requiring user activation; download is the fallback. Neither document content nor editor selection is modified.

## Verification boundaries

Go integration tests cover authorization and native OneNote/Butterfly round trips; bounded-process tests cover output-limit bypasses. Frontend tests exercise binary framing, expansion rejection, and native Goodnotes round trips. Playwright covers native import/export, bulk error/retry behavior, PDF placement/exports, and screenshot PNG dimensions/clipboard. The hardened-container test exports/imports a real native OneNote package.

The two private notebook samples are opt-in local checks. Goodnotes browser coverage accepts `MINDFLOW_GOODNOTES_FIXTURE=/path/to/file.goodnotes`; it is never required by CI. Test fixtures under `docs/examples` and the small unit-test ZIP are synthetic. Destination app editing/visual fidelity requires the [manual checklist](../user-docs/interoperability-testing.md); Goodnotes and OneNote are not available on CI runners.
