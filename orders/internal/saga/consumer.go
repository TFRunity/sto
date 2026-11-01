package saga

// TODO(dev): обработчик очереди orders.saga_replies.
//   - StockDeducted / StockRejected {saga_id, order_id}               → ADR-025 (А-07)
//   - ack только после коммита транзакции репозитория                → ADR-025
