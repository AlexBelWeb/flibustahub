// Package migrations embeds catalog schema files.
package migrations

import "embed"

// FS holds numbered SQL migration files.
//
//go:embed *.sql
var FS embed.FS
