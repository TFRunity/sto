package saga

// TODO(dev): воркер дедлайнов.                                       → ADR-030, D-02
//   - тик ~1 с; просроченные verifying_stock
//   - одна транзакция: условный переход в failed, paid → active,
//     outbox CancelStock {saga_id}
//   - конкурирует с консьюмером ответов — только условный UPDATE, без mutex
