package account

import _ "embed"

// Schema defines the storage owned by account.
//
//go:embed schema.sql
var Schema string
