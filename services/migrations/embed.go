package migrations

import "embed"

// FS содержит SQL-миграции Services.
//
//go:embed *.sql
var FS embed.FS
