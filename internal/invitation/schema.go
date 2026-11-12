package invitation

import _ "embed"

// Schema defines the storage owned by invitation.
//
//go:embed schema.sql
var Schema string
