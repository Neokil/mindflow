# Email artwork

`icon.png` is the application mark embedded beside the mindflow name in email headers. Keep it byte-for-byte synchronized with [`web/public/icon.png`](../../../web/public/icon.png).

`github.png` is a transparent, 2× raster rendering of the same Lucide GitHub icon used in the application header. It is embedded as a CID image so email clients do not need SVG support or a remote image request. Lucide's ISC/Feather notices are included in [THIRD_PARTY_NOTICES.md](../../../THIRD_PARTY_NOTICES.md).

Both images are embedded as CID attachments by `internal/email/message.go`; email clients do not need to fetch remote artwork. The GitHub icon links to the repository. See [branding](../../../docs/developer-docs/branding.md) for palette and email layout guidance.
