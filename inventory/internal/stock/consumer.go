package stock

// TODO(dev): обработчик очереди inventory.saga_commands.
//   - DeductStock {saga_id, order_id, items[{item_id, qty}]},
//     CancelStock {saga_id, order_id}                                → ADR-025 (А-07)
//   - ack только после коммита транзакции репозитория                → ADR-025
