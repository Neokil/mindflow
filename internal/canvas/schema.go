package canvas

import _ "embed"

// Schema defines the storage owned by canvas.
//
//go:embed schema.sql
var Schema string
