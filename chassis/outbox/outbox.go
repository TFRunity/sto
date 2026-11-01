// Package outbox — transactional outbox (ADR-005, D-01).
package outbox

// TODO(dev): outbox.
//   - вставка сообщения в переданной tx (DBTX) — в одной транзакции
//     с доменными данными                                           → ADR-024, ADR-005
//   - payload — только идентификаторы, без персональных данных       → D-01 (сквозное)
//   - воркер: тик → SELECT пачки WHERE published = false → публикация
//     с confirm на каждое сообщение → UPDATE … WHERE id = ANY($1)    → D-01
//   - без FOR UPDATE SKIP LOCKED (один экземпляр сервиса)            → D-01, ADR-006
//   - AMQP MessageId = outbox.id; при nack строка ждёт следующего тика
// Таблица outbox и частичный индекс WHERE published = false — в миграциях сервисов.
