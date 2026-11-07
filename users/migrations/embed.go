// Package migrations — SQL-миграции Users, встроенные в бинарник (ADR-024).
package migrations

import "embed"

// FS — все *.sql этого каталога. Читает postgres.Migrate.
//
//go:embed *.sql
var FS embed.FS
