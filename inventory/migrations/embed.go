package migrations

import "embed"

// FS содержит SQL-миграции Inventory, включая seed товаров.
//
//go:embed *.sql
var FS embed.FS
