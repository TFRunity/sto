// Package item — товары каталога: цена по id для Orders (ADR-034).
package item

// TODO(dev): чтение товара.
//   - GET цены товара по id — Orders фиксирует её в позиции заказа    → ADR-034, sequence-add-items
//   - деньги — decimal, в JSON строкой                               → ADR-026
//   - deleted_at IS NULL                                             → ADR-033
// CRUD каталога — неделя 2; товары недели 1 заводит seed-миграция.
