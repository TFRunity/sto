package stock

// TODO(dev): репозиторий попыток (одна транзакция на команду).       → ADR-031, D-03
//   - stock_attempts (PK saga_id) + stock_attempt_items              → ADR-030
//   - списание: UPDATE … SET quantity = quantity - N
//     WHERE id = ? AND quantity >= N; хоть одна позиция не прошла →
//     откат списания, запись rejected                                → ADR-030, D-09
//   - возврат по deducted → cancelled
//   - ответ StockDeducted / StockRejected — в outbox той же tx        → ADR-005
