package session

import _ "embed"

// Schema defines the storage owned by session.
//
//go:embed schema.sql
var Schema string
