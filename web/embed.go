package web

import "embed"

// Files contains the compiled frontend. Run npm run build before a release build.
//
//go:embed all:dist
var Files embed.FS
