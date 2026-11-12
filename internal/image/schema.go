package image

import _ "embed"

// Schema defines the storage owned by image.
//
//go:embed schema.sql
var Schema string
