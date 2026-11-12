package workspace

import _ "embed"

// Schema defines the storage owned by workspace.
//
//go:embed schema.sql
var Schema string
