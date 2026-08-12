// Package migrations embeds the poller's SQL migration files so the
// binary carries its own schema.
package migrations

import "embed"

// Files holds every *.sql file in this directory, applied in filename
// order by internal/migrate.
//
//go:embed *.sql
var Files embed.FS
