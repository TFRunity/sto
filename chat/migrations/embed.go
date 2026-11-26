package migrations

import "embed"

// FS содержит SQL-миграции Chat.
//
//go:embed *.sql
var FS embed.FS
