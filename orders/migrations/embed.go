package migrations

import "embed"

// FS содержит SQL-миграции Orders.
//
//go:embed *.sql
var FS embed.FS
