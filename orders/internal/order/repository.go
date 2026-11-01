package order

// TODO(dev): репозиторий заказов (метод на сценарий, tx внутри).      → ADR-031, D-03
//   - создание заказа инженером сразу active + order_engineers,
//     одна транзакция                                                → sequence-create-order, ADR-020
//   - добавление позиции с зафиксированной ценой                     → ADR-034, sequence-add-items
//   - чтение заказа с позициями и признаком «оплата подтверждена»
//     (сага completed)                                               → ADR-030, TODO.md §7
