// Package migrations embeds workspace's independently versioned Goose migrations.
package migrations

import "embed"

//go:embed *.sql
var files embed.FS

// Files returns the embedded SQL files consumed by Goose.
func Files() embed.FS { return files }
