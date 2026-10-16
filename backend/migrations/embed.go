// Package migrations embeds the numbered golang-migrate SQL files so the
// server binary and cmd/migrate carry their schema with them (nothing needs
// to be copied into the container image).
//
// Conventions:
//   - Files are NNNNNN_description.up.sql / NNNNNN_description.down.sql.
//   - Migrations are additive and append-only: never edit a migration that
//     has been applied to a shared database; write a new one instead.
//   - 000001_initial_schema is the baseline snapshot of the former
//     001_schema.sql.
//   - Dev-only seed data lives in backend/seeds/ and is NOT a migration.
package migrations

import "embed"

// FS holds every *.up.sql and *.down.sql file in this directory.
//
//go:embed *.up.sql *.down.sql
var FS embed.FS
