package saga

// TODO(dev): репозиторий саги (каждый сценарий — одна транзакция).    → ADR-031, D-03
//   - старт: UPDATE orders SET status='paid' WHERE status='active'
//     + saga_state (deadline = now() + 10s) + outbox DeductStock      → ADR-030, sequence-pay-order
//   - StockDeducted: условный UPDATE … WHERE current_step =
//     'verifying_stock' → completed + INSERT certificates pending     → ADR-030, ADR-035
//   - StockRejected: → failed + компенсация paid → active
//   - 0 затронутых строк — не ошибка: другой участник успел раньше    → D-02 (сквозное)
