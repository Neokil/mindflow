package passwordreset

import _ "embed"

// Schema defines the storage owned by passwordreset.
//
//go:embed schema.sql
var Schema string
