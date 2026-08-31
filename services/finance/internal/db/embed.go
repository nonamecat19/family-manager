// Package dbfs embeds this service's migrations so a container ships with its schema and
// applies it at boot — no separate migrate step in the compose file, no drift between the
// binary and the SQL it expects.
//
// The embed lives here, next to the .sql files, because go:embed cannot reach through "..".
package dbfs

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the path of the embedded directory inside Migrations.
const MigrationsDir = "migrations"
