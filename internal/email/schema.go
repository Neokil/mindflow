package email

import _ "embed"

// Schema defines the storage owned by email.
//
//go:embed schema.sql
var Schema string
