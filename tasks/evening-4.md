# Вечер 4 — задачи

**Итог вечера:** строка, вставленная в `outbox` Orders, сама уходит в RabbitMQ, приходит в
очередь `inventory.saga_commands`, и консьюмер Inventory её принимает (лог + ack); в
`outbox` строка помечена `published = true`. То же в обратную сторону — Inventory →
`orders.saga_replies` → Orders.

Задачи идут по порядку — каждая опирается на предыдущие. После каждой — ревью.
Отмечай сделанное: `[ ]` → `[x]`.

Это вечер «труб»: бизнес-смысла в сообщениях пока нет, обработчики — заглушки «залогировал
и подтвердил». Смысл появится в вечере 5 (сага).

Топология уже готова: exchange, очереди и binding описаны в definitions брокера
(`docker-compose.yml`, `configs: rabbit_definitions`, ADR-025, А-07). Сервисы их **не
создают**.

| Откуда | Exchange | Routing key = очередь | Сообщения |
|---|---|---|---|
| Orders | `saga_commands` | `inventory.saga_commands` | `DeductStock`, `CancelStock` |
| Inventory | `saga_replies` | `orders.saga_replies` | `StockDeducted`, `StockRejected` |

---

## [ ] 1. Публикатор с подтверждениями

Нужно написать в `chassis/rabbit/publisher.go` публикатор, который отправляет одно сообщение
и ждёт подтверждения брокера, чтобы outbox помечал строку отправленной только когда брокер
её точно принял (ADR-025, ADR-005).
Используя `github.com/rabbitmq/amqp091-go`: `amqp.Dial`, `Connection.Channel`,
`Channel.Confirm(false)`, `Channel.PublishWithDeferredConfirmWithContext`,
`DeferredConfirmation.WaitContext`, `amqp.Publishing` (`MessageId`, `Type`, `ContentType`,
`DeliveryMode: amqp.Persistent`, `Body`).

Отдельное соединение под публикацию; один канал — один владелец (канал не потокобезопасен).
`MessageId` = `outbox.id`, `Type` = `event_type`. `WaitContext` вернул `false` (nack) —
ошибка, строка подождёт следующего тика.

**Проверка:** в задаче 5. Тестов нет: обвязка (ADR-013).

**Подумай:** что вернёт публикация в exchange без подходящего binding, если `mandatory =
false`, — ack или nack? Нужно ли нам это ловить сейчас.

---

## [ ] 2. Outbox: вставка в чужой транзакции

Нужно написать в `chassis/outbox` функцию «добавить сообщение», которая пишет строку в
`outbox` **той транзакцией, которую ей передали**, чтобы событие и доменные данные
сохранялись атомарно (ADR-005, ADR-024).
Используя минимальный свой интерфейс `DBTX` с методом `Exec(ctx, sql, args...)
(pgconn.CommandTag, error)` — ему подходят и `pgx.Tx`, и `*pgxpool.Pool`; `json.Marshal`
для `payload`.

В `payload` — только идентификаторы и технические значения (количества), без имён,
телефонов, email (D-01, А-08). Id строки — `uuid.New()` в Go (ADR-026).

**Проверка:** в задаче 5 — тестовая вставка через эту функцию появляется в таблице.
Тестов нет: SQL без правил.

**Подумай:** почему функция принимает `DBTX`, а не `*pgxpool.Pool` — что сломается, если
она сама откроет транзакцию?

---

## [ ] 3. Outbox: воркер

Нужно написать в `chassis/outbox` воркер, который раз в тик забирает пачку
неопубликованных строк, публикует их по одной и помечает отправленные, чтобы событие
гарантированно доходило до брокера, даже если он лежал (ADR-005, D-01).
Используя `time.NewTicker` + `select` по `ctx.Done()`, `pool.Query` +
`pgx.CollectRows(rows, pgx.RowToStructByName[T])`, публикатор из задачи 1.

- `SELECT … WHERE published = false ORDER BY created_at LIMIT N` — частичный индекс уже есть;
- публикация каждой строки с ожиданием confirm;
- `UPDATE outbox SET published = true, published_at = now() WHERE id = ANY($1)` — только
  подтверждённые;
- **без** `FOR UPDATE SKIP LOCKED` — экземпляр один (D-01);
- ошибка публикации — warn-лог, строка остаётся на следующий тик; воркер не падает.

Exchange и routing key у сервиса один (таблица выше) — передай их воркеру при создании.

**Проверка:** в задаче 5. Тестов нет: обвязка (ADR-013).

**Подумай:** брокер упал на середине пачки: три строки ушли, четвёртая — нет. Что окажется в
базе и что получит получатель после перезапуска? Почему это допустимо (at-least-once +
`saga_id`, ADR-030).

---

## [ ] 4. Консьюмер

Нужно написать в `chassis/rabbit/consumer.go` консьюмер, который читает свою очередь по одному
сообщению и подтверждает его только после успешной обработки, чтобы падение сервиса не
теряло сообщений (ADR-025).
Используя `Channel.QueueDeclarePassive` (проверка, что очередь есть), `Channel.Qos(1, 0,
false)`, `Channel.ConsumeWithContext` с `autoAck = false`, `Delivery.Ack(false)`,
`Delivery.Nack(false, true)`, заголовок `Delivery.Headers["x-delivery-count"]`.

- Обработчик — функция, которую передаёт сервис; вернула `nil` → `Ack` (сервис к этому
  моменту уже закоммитил транзакцию);
- вернула ошибку → пауза 0,2 с → `Nack` с requeue;
- `x-delivery-count >= 5` → error-лог с `MessageId` и `Type` → `Ack` (выбросить);
- очереди нет → ошибка старта, сервис падает, compose перезапустит (ADR-038);
- соединение оборвалось → выход с ошибкой, процесс завершается. Супервизор реконнекта —
  первая линия отреза (`PLAN.md`), неделя 2.

**Проверка:** в задаче 5. Тестов нет: обвязка (ADR-013).

**Подумай:** на первой доставке заголовка `x-delivery-count` нет — что тогда вернёт чтение из
`map[string]any`, и какого типа значение, когда он есть (`int64`?). Проверь, а не угадывай.

---

## [ ] 5. Сборка: трубы в Orders и Inventory

Нужно запустить в `main.go` обоих сервисов outbox-воркер и консьюмер своей очереди рядом с
HTTP-сервером, чтобы всё останавливалось вместе по `Ctrl+C` (ADR-038).
Используя задачи 1–4, `sync.WaitGroup.Go` (или `golang.org/x/sync/errgroup`), общий `ctx` из
`signal.NotifyContext` (вечер 1, задача 4).

- Orders: воркер → `saga_commands`; консьюмер `orders.saga_replies` — заглушка «info-лог +
  `nil`»;
- Inventory: воркер → `saga_replies`; консьюмер `inventory.saga_commands` — та же заглушка.
- Ошибка любой из частей → остановка остальных и выход с ненулевым кодом.

**Проверка (итог вечера):** `make up` (с «можно») или `make run` обоих сервисов, затем:
- `docker compose exec orders-db psql …` → `INSERT INTO outbox (aggregate_id, event_type,
  payload) VALUES (gen_random_uuid(), 'DeductStock', '{}')` → через ~1 с в логе Inventory
  строка о сообщении, в `outbox` у строки `published = true`;
- то же в `inventory-db` → сообщение в логе Orders;
- `make stop SVC=rabbitmq` → вставка → строка висит `published = false`, воркер пишет warn,
  но не падает; брокер обратно → строка уходит;
- `Ctrl+C` — сервисы выходят с кодом 0.

**Подумай:** HTTP-сервер, воркер и консьюмер ждут один и тот же `ctx`. В каком порядке их
останавливать, чтобы не потерять ack обрабатываемого сообщения?

---

## Подключаемые библиотеки
Каждую подключаем, когда дойдёшь до задачи, — команду даю я, запуск с твоего «можно».

| Задача | Библиотека |
|---|---|
| 1 | `rabbitmq/amqp091-go` в `chassis` |
| 5 | `golang.org/x/sync` (если выберешь `errgroup`) в Orders и Inventory |

## Что сознательно не делаем (сначала работает)
- Супервизор реконнекта RabbitMQ — линия отреза 1: при обрыве сервис падает,
  `restart: unless-stopped` поднимает его заново. Фиксируется в `TODO.md`.
- Dead-letter exchange — отклонён в ADR-025; выбрасываем после 5 доставок.
- Чистка опубликованных строк outbox — не нужна (D-01, «журнал без чистки»).

Оценка — 3–3,5 часа.
