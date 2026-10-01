// Package migrations embeds the goose SQL migrations in the server binary.
// Migrations are forward-only (ADR-0014): add a new file, never edit or roll
// back an applied one, and write no "+goose Down" section.
package migrations

import "embed"

// FS holds every migration file, named NNNNN_description.sql.
//
//go:embed *.sql
var FS embed.FS
