package main

// TODO(dev): сборка Orders.
//   - конфиг (chassis/config): HTTP_ADDR, DATABASE_URL, RABBITMQ_URL,
//     INVENTORY_URL, USERS_URL, GOTENBERG_URL
//   - pgxpool + адаптер decimal в AfterConnect                       → ADR-026
//   - миграции goose при старте                                      → ADR-024
//   - HTTP, консьюмер orders.saga_replies, outbox-воркер,
//     воркер дедлайнов, воркер справок
//   - graceful shutdown всего перечисленного по отмене ctx
func main() {
	panic("TODO(dev): orders")
}
