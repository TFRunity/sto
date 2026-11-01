package main

// TODO(dev): сборка Inventory.
//   - конфиг (chassis/config): HTTP_ADDR, DATABASE_URL, RABBITMQ_URL
//   - pgxpool + адаптер decimal в AfterConnect                       → ADR-026
//   - миграции goose при старте                                      → ADR-024
//   - HTTP (цены), консьюмер inventory.saga_commands, outbox-воркер
//   - graceful shutdown: HTTP, консьюмер и воркер — по отмене ctx
func main() {
	panic("TODO(dev): inventory")
}
