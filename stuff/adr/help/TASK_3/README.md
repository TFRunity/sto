# Вечер 4. Запуск и проверка RabbitMQ

## Что запускалось

| Команда | Зачем |
|---|---|
| `make test` | Прогоняет существующие Go-тесты и компилирует пакеты сервисов. |
| `bash /tmp/sto-evening-4.sh` | Разовый прогон на поднятом Compose. Дважды возвращает уже обработанную команду в очередь публикации с тем же `saga_id`. |
| `make stop SVC=rabbitmq` | Останавливает брокер перед повторной постановкой команды в outbox. |
| `docker compose start rabbitmq` | Возвращает брокер, чтобы проверить отправку накопившейся команды. |
| `docker stop --signal SIGINT --time 15 "$orders_id" "$inventory_id"` | Отправляет процессам Orders и Inventory тот же сигнал, что Ctrl+C. Через 15 секунд Docker мог бы принудительно остановить процесс. |
| `docker compose start orders inventory` | Снова запускает оба сервиса после проверки штатного выхода. |

Чтобы запустить повторную отправку, прогон выставляет `published = false` у существующей строки `DeductStock`. Это команда уже завершённой саги, поэтому повтор должен породить ещё один `StockDeducted` с тем же `saga_id` без повторного списания. SQL-операция эквивалентна шагу из [RUNNING.md](../../../RUNNING.md), раздел «Повторно отправить сообщение из outbox».

## Какие функции запускают работу в Go

1. `main` в Orders и Inventory создаёт отменяемый контекст через `signal.NotifyContext` для SIGINT и SIGTERM и вызывает `run`. [Orders](../../../orders/cmd/orders/main.go) · [Inventory](../../../inventory/cmd/inventory/main.go)
2. `run` готовит PostgreSQL через `postgres.ConnectAndMigrate`. Затем через `errgroup.WithContext` запускает `httpx.Run` и `outbox.RunRabbit`. Так ошибка одной части отменяет работу остальных. [Подключение к БД](../../../chassis/postgres/postgres.go) · [HTTP](../../../chassis/httpx/server.go)
3. `outbox.RunRabbit` вызывает `rabbit.Run`. Тот поднимает два соединения: `openPublisher` готовит отправку с подтверждениями, `openDeliveries` подписывается на очередь с ручным подтверждением. `runWorkers` запускает оба цикла. [Outbox](../../../chassis/outbox/outbox.go) · [RabbitMQ](../../../chassis/rabbit/link.go)
4. `publishBatch` вызывает `Publisher.Publish`. Воркер помечает запись отправленной после подтверждения RabbitMQ. На другой стороне `handleDelivery` вызывает обработчик сервиса. `Ack` отправляется после успешного возврата обработчика, то есть после коммита его транзакции. [Отправка](../../../chassis/rabbit/publisher.go) · [Доставка](../../../chassis/rabbit/consumer.go)

## Результаты

- `make test` прошёл. Выполнились существующие тесты Users. Пакеты RabbitMQ, outbox, Orders и Inventory прошли компиляцию.
- При работающем RabbitMQ команда дошла до Inventory, а ответ дошёл до обработчика Orders. Количество опубликованных ответов для выбранной саги увеличилось с 2 до 3. Остаток сохранился на 18 единицах.
- При выключенном RabbitMQ запись осталась с `published = false`. После запуска брокера прошли команда и ответ, счётчик ответов увеличился с 3 до 4. Остаток сохранился. Orders и Inventory перезапускались по политике Compose, счётчики обоих контейнеров выросли с 0 до 7. Это поведение текущей схемы восстановления при потере соединения.
- Orders и Inventory завершились с кодом 0 на SIGINT. После запуска оба контейнера стали `healthy`. RabbitMQ тоже оставлен включённым.
- Сообщение `ignoring stock reply` ожидаемо на этом прогоне: Orders получил ответ для уже завершённой саги и оставил её состояние прежним.
