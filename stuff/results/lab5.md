# Лабораторная работа № 5. Запросы к базе заказов СТО

Три отчёта соответствуют операциям, которые можно показать на экранах клиента, инженера и директора. Для лабы 6 используйте один и тот же UUID заказа на скриншотах интерфейса и в результатах SQL. [Порядок действий и экраны](lab4.md).

## Данные и подключение

1. В DBeaver подключитесь к PostgreSQL по адресу `127.0.0.1:5434`, база `orders`. Имя пользователя и пароль берите из `ORDERS_DB_USER` и `ORDERS_DB_PASSWORD` в локальном `.env`. [Настройки Compose](../docker-compose.yml#L218).
2. По сценариям 01–04 [лабы 4](lab4.md) зарегистрируйте клиента, добавьте автомобиль и отправьте заявку. Сохраните UUID заказа и автомобиля из адресной строки. До первого запуска запросов инженер оставляет заявку в пуле.
3. Вставляйте запросы в SQL-редактор DBeaver и заменяйте текст в угловых скобках нужным UUID, сохраняя одинарные кавычки. Запускайте блоки по одному и снимайте таблицы результатов. Действия в интерфейсе создадут и изменят данные, все блоки ниже читают их через `SELECT`.

### Модель данных для диаграммы DBeaver

| Таблица | Роль в сценарии |
| --- | --- |
| `orders` | Заявка и заказ хранятся в одной строке. Назначение меняет `status` и `engineer_id`, UUID сохраняется. |
| `order_engineers` | Хранит исполнителей заказа. Пара `(order_id, engineer_id)` уникальна, `order_id` ссылается на `orders.id`. |
| `order_items` | Товары заказа. Сумма позиции равна `quantity * price`. |
| `order_services` | Услуги, добавленные к заказу. Сумма позиции равна `quantity * price`. |
| `saga_state` | Состояние обработки оплаты, используется в истории для признака подтверждённой оплаты. |

В последних трёх таблицах `order_id` также ссылается на `orders.id`. Идентификаторы клиента и инженера относятся к сервису Users, а идентификатор автомобиля — к Vehicle. Базы этих сервисов отдельные, поэтому в диаграмме БД Orders у этих полей нет внешних ключей. [Миграции Orders](../orders/migrations/).

### Подготовка параметров

Подставьте UUID из адреса клиентской карточки. Из результата возьмите `client_id`, а `vehicle_id` сверьте с адресом карточки ТС. После выполнения сценария 04 ожидаются `status = requested` и пустой `engineer_id`.

```sql
SELECT id, client_id, vehicle_id, status, engineer_id
FROM orders
WHERE id = '<UUID_ЗАКАЗА>'::uuid;
```

## Запрос 1. Какие заказы видит клиент

**Экран** — «Мои заказы» после отправки заявки, сценарий 04. Запрос отбирает заказы текущего клиента и сортирует их от новых к старым. Для сверки с разделом «История работ» конкретного автомобиля замените `NULL::uuid` в CTE на `'<UUID_ТС>'::uuid`. Признак `payment_confirmed` учитывает дату оплаты и завершённую сагу.

```sql
WITH params AS (
    SELECT '<UUID_КЛИЕНТА>'::uuid AS client_id,
           NULL::uuid AS vehicle_id
)
SELECT o.id, o.vehicle_id, o.status, o.stage, o.created_at,
       (o.paid_at IS NOT NULL OR EXISTS (
           SELECT 1
           FROM saga_state AS s
           WHERE s.order_id = o.id AND s.current_step = 'completed'
       )) AS payment_confirmed
FROM orders AS o
CROSS JOIN params AS p
WHERE o.client_id = p.client_id
  AND (p.vehicle_id IS NULL OR o.vehicle_id = p.vehicle_id)
ORDER BY o.created_at DESC, o.id DESC;
```

**Результат для отчёта** — видна строка с UUID демонстрационной заявки, статусом `requested` и `payment_confirmed = false`. Если повторить запрос после назначения инженера, у той же строки будет `active`. [Запрос приложения ListClientOrderHistory](../orders/queries/orders.sql#L50).

## Запрос 2. Какие заявки доступны инженеру

**Экран** — «Пул заявок» на Android, сценарии 05–06. Запрос выбирает только заявки со статусом `requested`, от самых ранних к поздним. Услуга, дата и комментарий помогают найти нужную строку на экране.

```sql
SELECT id, client_id, vehicle_id, requested_service, desired_date, comment, created_at
FROM orders
WHERE status = 'requested'
ORDER BY created_at, id;
```

**Результат для отчёта** — до нажатия «Взять в работу» есть строка с нужным UUID, после нажатия и повторного выполнения запроса её нет. Снимите оба результата. [Запрос приложения ListServiceBookings](../orders/queries/orders.sql#L36).

### Запрос 2а. Что изменилось после «Взять в работу»

Нажатие вызывает `ClaimOrder` и `AddOrderEngineer` в одной транзакции. Первый запрос ставит `active` и записывает ответственного инженера, второй добавляет его в список исполнителей. Этот контрольный `SELECT` проверяет обе записи по одному UUID заказа.

```sql
SELECT o.id, o.status, o.stage, o.engineer_id,
       EXISTS (
           SELECT 1
           FROM order_engineers AS oe
           WHERE oe.order_id = o.id AND oe.engineer_id = o.engineer_id
       ) AS engineer_in_team
FROM orders AS o
WHERE o.id = '<UUID_ЗАКАЗА>'::uuid;
```

**Результат для отчёта** — `status = active`, `stage = принят`, `engineer_in_team = true`. Для инженера из тестовой миграции `engineer_id = 00000000-0000-4000-8000-000000000001`. [SQL назначения](../orders/queries/orders.sql#L19), [транзакция в репозитории](../orders/internal/order/repository.go#L189).

## Запрос 3. Какие заказы видит директор с выбранным фильтром

**Экран** — «Все заказы», сценарий 08. Для случая «В работе» и поиска по госномеру используйте UUID этого автомобиля. В приложении Gateway сначала получает совпавшие UUID по полю поиска, затем Orders фильтрует заказы по статусу и этим UUID. Для сверки по одному автомобилю ниже используется уже известный UUID. Суммы товаров и услуг вычисляются по сохранённым ценам позиций, пустая сумма равна нулю. Запрос возвращает до 20 строк, как первая страница в вебе.

```sql
WITH page AS (
    SELECT id, client_id, vehicle_id, engineer_id, status, stage, created_at
    FROM orders
    WHERE status = 'active'
      AND vehicle_id = '<UUID_ТС>'::uuid
    ORDER BY created_at DESC, id DESC
    LIMIT 20
)
SELECT p.*,
       (COALESCE((SELECT SUM(quantity * price) FROM order_items WHERE order_id = p.id), 0)
        + COALESCE((SELECT SUM(quantity * price) FROM order_services WHERE order_id = p.id), 0))::numeric AS total
FROM page AS p
ORDER BY p.created_at DESC, p.id DESC;
```

**Результат для отчёта** — тот же UUID заказа, ответственный инженер, `active` и сумма из «Сводки заказа». Если позиции ещё не добавлялись, сумма равна нулю. Для проверки фильтра «Заявка» замените `active` на `requested` и убедитесь, что конкретный UUID исчез. Имена и госномер в веб-таблицу добавляет Gateway, а результат SQL показывает UUID. [Запрос приложения ListOrdersForDirector](../orders/queries/director.sql#L1), [агрегация в Gateway](../gateway/internal/director.go#L99).

## Что приложить к отчёту

- Диаграмму пяти таблиц из DBeaver.
- Тексты запросов 1, 2, 2а и 3 и таблицы с фактическими результатами.
- Для совмещённой лабы 6 — экран «Мои заказы» рядом с запросом 1, пул инженера до и заказ после назначения рядом с запросами 2 и 2а, «Все заказы» и «Сводку заказа» рядом с запросом 3. На всех шагах сверьте UUID заказа или его видимый префикс.
