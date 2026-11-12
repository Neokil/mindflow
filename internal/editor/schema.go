package editor

import _ "embed"

// Schema defines the storage owned by editor.
//
//go:embed schema.sql
var Schema string
