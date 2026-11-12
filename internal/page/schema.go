package page

import _ "embed"

// Schema defines the storage owned by page.
//
//go:embed schema.sql
var Schema string
