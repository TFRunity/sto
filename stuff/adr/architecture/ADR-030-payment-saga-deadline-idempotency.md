# ADR-030: Сага оплаты — ключ `saga_id`, дедлайн 10 с, идемпотентный Inventory

## Статус
Принято. Частично заменяет ADR-006. Ключ `saga_id` — **пересматриваемо**.

В v3 старт и компенсация оплаты заменены на [ADR-043](ADR-043-manual-payment-stock-gate.md). Правила идемпотентности Inventory используются и в v3.

## Контекст
Доставка RabbitMQ — at-least-once, а консьюмер выбрасывает сообщение
после 5 доставок (ADR-025). Ответ Inventory может не прийти никогда, и
тогда сага навсегда остаётся в `verifying_stock`, заказ навсегда `paid`, а
списаны ли товары — неизвестно. Дубль команды «спиши» без защиты спишет
остаток дважды (ADR-006 откладывал идемпотентность до появления «шага с
дорогим побочным эффектом» — списание им и является).

## Решение

### Старт (одна транзакция в Orders)
`UPDATE orders SET status='paid' WHERE id=? AND status='active'` +
`INSERT saga_state (saga_id, order_id, current_step='verifying_stock',
deadline = now() + interval '10 seconds')` + outbox `DeductStock
{saga_id, order_id, items[{item_id, qty}]}`. Клиенту — `202 Accepted`
— клиент перечитывает статус заказа, UI показывает «оплата
подтверждается» (разбор А-06, 30.10.2026: принято; отклонено удержание
запроса до завершения саги — до 10 с висящего соединения, а опрос
статуса всё равно нужен на случай обрыва).

### Ключ идемпотентности — `saga_id`
Каждая попытка оплаты — отдельная сага со своим id. Команды и ответы
несут `saga_id` (+ `order_id` для логов). Tombstone одной попытки не
блокирует следующую.

### Inventory
- Таблица попыток `stock_attempts` (PK `saga_id`) в состояниях
  `deducted | rejected | cancelled` и таблица позиций
  `stock_attempt_items` (что и сколько списано). Inventory — сам
  источник истины для возврата (Решение А-07 (принято 30.10.2026): имена таблиц).
- Списание: по каждой позиции `UPDATE inventory SET quantity = quantity
  - N WHERE id = ? AND quantity >= N` + `CHECK (quantity >= 0)`. Если
  хоть одна позиция не прошла — откат всего списания, запись `rejected`,
  ответ `StockRejected`.
- `rejected` записывается всегда, чтобы поздний дубль команды не списал.
- Повторное `DeductStock` → **повторяется тот же ответ** (не молчаливый
  ack: потерянный первый ответ иначе привёл бы к ложному откату).
- `CancelStock`: `deducted` → вернуть позиции, состояние `cancelled`;
  попытки нет → записать **tombstone** `cancelled`; `rejected` /
  `cancelled` → ничего не делать. Последующее `DeductStock` по
  `cancelled` отклоняется — схема не зависит от порядка доставки.
- Ответы Inventory публикует через свой outbox (ADR-005).

### Orders
- Переходы саги — **условный `UPDATE … WHERE current_step =
  'verifying_stock'`** с проверкой числа строк: консьюмер ответов и
  воркер дедлайнов конкурируют за одну сагу даже в одном процессе.
  0 строк — не ошибка, а «другой участник успел раньше».
- Ответ `StockDeducted` → `completed`; заказ остаётся `paid`; в той же
  транзакции создаётся справка `pending` (ADR-035).
- Ответ `StockRejected` → `failed` + компенсация `UPDATE orders SET
  status='active' WHERE id=? AND status='paid'`.
- **Воркер дедлайнов** (тик ~1 с): просроченные `verifying_stock` → в
  одной транзакции условный переход в `failed`, `paid → active`, outbox
  `CancelStock {saga_id}`.
- Ответ на неизвестную сагу или пройденный шаг → ack + warn-лог.
- Потерянная команда `CancelStock` (выброшена после 5 доставок) →
  error-лог с `saga_id`, ручной разбор через SQL. Риск принят: двойной
  сбой при масштабе СТО.
- Автоматического повтора оплаты нет (ADR-016).

### Схема `saga_state` (заменяет SQL из ADR-006)
```sql
CREATE TABLE saga_state (
    saga_id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     uuid NOT NULL REFERENCES orders(id),
    current_step text NOT NULL
                 CHECK (current_step IN ('verifying_stock','completed','failed')),
    deadline     timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_saga_state_order_id ON saga_state (order_id);
CREATE INDEX idx_saga_state_deadline ON saga_state (deadline)
    WHERE current_step = 'verifying_stock';
```

> Решение А-05 (принято 30.10.2026): колонка `status` из ADR-006 удалена. Рядом с
> `current_step` её смысл не был определён, а два поля состояния — повод
> для рассинхронизации. Единственное состояние саги — `current_step`.

### Связанные правила
- «Выдан» доступен, а справка создаётся только при успешно завершённой саге
  (`EXISTS saga_state WHERE order_id=? AND current_step='completed'`):
  ADR-021, ADR-035. Одна сага на заказ может завершиться успехом не
  больше одного раза: после неё заказ остаётся `paid` навсегда.

## Альтернативы, которые рассматривались
- **Ключ `order_id`** — ломается на повторной оплате (tombstone первой
  попытки блокирует вторую) и на позднем дубле после `rejected`.
- **Mutex в Go вместо условного UPDATE** — защита в памяти для состояния
  в БД; не видит ручных правок и ломается при втором экземпляре.
- **Ручной разбор просроченных саг** — застрявшие заказы и человек,
  который должен за ними следить.
- **Часовая задача переотправки застрявших саг** — больше не нужна:
  застрять сага не может.
- **Дедупликация по `MessageId`** — ловит только повторную доставку того
  же сообщения; бизнес-ключ надёжнее.

## Последствия
**Плюсы:** сага всегда завершается за ≤ 10 с + тик; компенсация
безопасна при любом порядке и числе доставок.

**Минусы:** ложный откат возможен при задержке > 10 с (ответ «успех»
после дедлайна отбрасывается, списание отменяется командой
`CancelStock`); вторая таблица и логика в Inventory.

## Дата
2026-10-30
