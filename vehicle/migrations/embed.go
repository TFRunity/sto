package migrations

import "embed"

// FS содержит SQL-миграции Vehicle.
//
//go:embed *.sql
var FS embed.FS
