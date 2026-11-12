// Package migrations embeds image's independently versioned Goose migrations.
package migrations

import "embed"

//go:embed *.sql
var files embed.FS

// Files returns the embedded SQL files consumed by Goose.
func Files() embed.FS { return files }

// Dependencies identifies domain schemas required before these migrations run.
func Dependencies() []string { return []string{"account", "email"} }
