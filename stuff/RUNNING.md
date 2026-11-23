# Запуск, проверка и команды проекта (v1)

Шпаргалка по тому, как поднять проект, проверить, что он работает, и чем им управлять.
Источник правды по целям — шапка `Makefile`; по портам и сервисам — `docker-compose.yml`;
почему всё устроено так — ADR-036, ADR-038, ADR-040.

> Агенты запускают любые команды отсюда **только после «можно»** от разработчика
> (`AGENTS.md`, «Главное правило»). Разработчик запускает сам, когда хочет.

Пометки готовности: ✅ — работает уже сейчас; 🕒 — заработает после указанного вечера
(`tasks/evening-N.md`).

---

## 1. Что нужно на машине

| Инструмент | Зачем | Проверить |
|---|---|---|
| Go 1.27 | сборка, тесты, `make run` | `go version` |
| Docker + Docker Compose v2 | БД, RabbitMQ, Gotenberg, сервисы в контейнерах | `docker compose version` |
| make | все команды проекта | `make --version` |
| gcc | `sqlc` собирается с CGO (только для `make generate`) | `gcc --version` |
| curl | ручные проверки API | `curl --version` |
| jq (по желанию) | `make dev`, вытаскивать токен из ответа | `jq --version` |

`sqlc`, `oapi-codegen`, `golangci-lint` ставить не нужно — Makefile запускает их через
`go run pkg@version` с зафиксированной версией. Нет gcc — `sqlc` из Docker (см. шапку
`Makefile`, переменная `SQLC`).

---

## 2. Первая настройка (один раз)

1. Скопировать пример переменных и заменить пароли и секрет:
   ```sh
   cp .env.example .env
   openssl rand -base64 48      # → JWT_SECRET в .env
   ```
   `.env` в git не коммитится (ADR-011). Секрет `JWT_SECRET` общий у Users и Gateway.
2. Проверить, что `sto.localhost` резолвится в `127.0.0.1`: `curl -s http://sto.localhost/`
   после запуска nginx должен вернуть страницу приложения. Если имя не резолвится — строка
   `127.0.0.1 sto.localhost` в `/etc/hosts`.
3. Порт `80` (nginx) и порты `5433–5435`, `5672`, `3000` на хосте должны быть свободны.

---

## 3. Два режима запуска

### Режим А — всё в Docker (как в итоговой проверке) ✅

```sh
make up                         # собрать образы и поднять всё
make up SVC="gateway users"     # только эти сервисы + их зависимости (БД)
make ps                         # что запущено и healthy ли
make logs SVC=orders            # логи сервиса (Ctrl+C — выйти из логов)
make stop SVC=inventory         # остановить, не удаляя (сценарий дедлайна)
make down                       # остановить и удалить контейнеры (данные БД сохраняются)
```

Веб доступен на `http://sto.localhost/`, API на `http://sto.localhost/api/…` (ADR-038).
После правки `deploy/nginx/nginx.conf` запусти `make up SVC=nginx`.
Compose соберёт образ с новой конфигурацией и пересоздаст nginx.
Для разработки с горячей перезагрузкой установи зависимости командой
`npm ci --prefix frontend/web` и запусти `make web`. Открой
`http://localhost:3001`. Запросы к API идут через локальный nginx.
Android обращается к тому же Gateway через nginx. Адреса для эмулятора и
телефона приведены в [инструкции Android](../frontend/mobile/README.md).

### Режим Б — инфраструктура в Docker, сервис на хосте (для разработки) ✅

```sh
make infra                      # только БД, RabbitMQ, Gotenberg
make run SVC=users              # go run с переменными из .env и адресами 127.0.0.1
make dev SVC=users              # то же, LOG_LEVEL=debug, логи через jq
```

Удобно, когда правишь один сервис: перезапуск — `Ctrl+C` и снова `make run`. Соседей тоже
запускай на хосте, каждого в своём терминале (`make run SVC=inventory` и т.д.).

Можно смешивать: например, `make up SVC=gateway` в Docker нельзя направить на Users на
хосте — для смешанного режима Gateway тоже запускай через `make run`.

---

## 4. Адреса и порты

| Что | Режим А (Docker) | Режим Б (`make run`) |
|---|---|---|
| Вход в API | `http://sto.localhost/api/…` (nginx → Gateway) | `http://127.0.0.1:8080/…` (Gateway) |
| Gateway | только внутри сети | `127.0.0.1:8080` |
| Users | только внутри сети | `127.0.0.1:8081` |
| Orders | только внутри сети | `127.0.0.1:8082` |
| Inventory | только внутри сети | `127.0.0.1:8083` |
| users-db | `127.0.0.1:5433` | `127.0.0.1:5433` |
| orders-db | `127.0.0.1:5434` | `127.0.0.1:5434` |
| inventory-db | `127.0.0.1:5435` | `127.0.0.1:5435` |
| RabbitMQ (AMQP) | `127.0.0.1:5672` | `127.0.0.1:5672` |
| Gotenberg | `127.0.0.1:3000` | `127.0.0.1:3000` |

Management UI RabbitMQ (`15672`) наружу не публикуется (ADR-012) — вместо него
`rabbitmqctl` (раздел 8).

У каждого сервиса есть `GET /healthz` → `200` («процесс жив», ADR-038).

---

## 5. Сборка, тесты, линт, генерация ✅

В [справочнике команд](adr/help/README.md) разобраны `go mod tidy`, цели Make и диагностика генерации.

```sh
make build                      # go build ./... во всех модулях
make test                       # go test ./... во всех модулях
make test MODULE=orders         # только один модуль
make lint                       # golangci-lint v2, конфиг .golangci.yml
make lint MODULE=gateway
make generate                   # sqlc + oapi-codegen по всем модулям
make generate MODULE=orders
```

`MODULE` — одно из: `chassis gateway users orders vehicle inventory services`.

Когда запускать `make generate`: после правки `api/openapi.yaml`, `*/queries/*.sql`,
`*/migrations/*.sql`, `sqlc.yaml`, `oapi-codegen.yaml`. Сгенерированный код коммитится
(ADR-028) — после генерации посмотри `git diff`.

Перед тем как отдать кусок на ревью: `make lint test` (оба зелёные).

Один тест, подробно:
```sh
cd orders && go test ./internal/order -run TestTotal -v
```

---

## 6. Ручная проверка API (curl)

Ниже — для режима А. В режиме Б замени `http://sto.localhost/api` на
`http://127.0.0.1:8080` (через Gateway) или на адрес сервиса напрямую (без Gateway —
только Users, у остальных закрытые маршруты требуют заголовков Gateway).

```sh
API=http://sto.localhost/api
```

### 6.1 Регистрация и вход ✅ (Users) / через Gateway 🕒 вечер 2–3

```sh
# регистрация клиента → 201 и токен
curl -s -X POST $API/register -H 'Content-Type: application/json' \
  -d '{"name":"Иван","phone":"+79990000001","password":"secret-123"}'

# повтор с тем же телефоном → 409 phone_taken

# вход клиентом, токен — в переменную
CLIENT=$(curl -s -X POST $API/login -H 'Content-Type: application/json' \
  -d '{"phone":"+79990000001","password":"secret-123"}' | jq -r .token)

# вход инженером из seed (users/migrations/00002_seed_staff.sql)
ENGINEER=$(curl -s -X POST $API/login -H 'Content-Type: application/json' \
  -d '{"phone":"+70000000001","password":"engineer-example"}' | jq -r .token)
```

Директор из seed: `+70000000002` / `director-example`.

### 6.2 Доступ через Gateway 🕒 вечер 2–3

```sh
curl -i $API/orders                                   # без токена → 401
curl -i $API/unknown -H "Authorization: Bearer $CLIENT"  # неизвестный путь → 403
curl -i $API/orders -H "Authorization: Bearer $CLIENT" \
  -H 'X-User-Role: director'                           # роль всё равно из токена
```

### 6.3 Заказ и позиции 🕒 вечер 3

```sh
# создать заказ (инженер); client_id и vehicle_id — из seed / регистрации
curl -s -X POST $API/orders -H "Authorization: Bearer $ENGINEER" \
  -H 'Content-Type: application/json' \
  -d '{"client_id":"<uuid клиента>","vehicle_id":"<uuid ТС>"}'

# добавить товар (id — из seed Inventory)
curl -s -X POST $API/orders/<order_id>/items -H "Authorization: Bearer $ENGINEER" \
  -H 'Content-Type: application/json' -d '{"item_id":"<uuid>","quantity":2}'

# прочитать заказ: позиции, сумма, «оплата подтверждена»
curl -s $API/orders/<order_id> -H "Authorization: Bearer $ENGINEER" | jq
```

Точные имена полей — в `api/openapi.yaml` (агент добавит их перед вечером 3).

### 6.4 Оплата и сага 🕒 вечер 5

```sh
curl -i -X POST $API/orders/<order_id>/pay -H "Authorization: Bearer $CLIENT"  # → 202
sleep 2
curl -s $API/orders/<order_id> -H "Authorization: Bearer $CLIENT" | jq          # paid, подтверждена
```

### 6.5 Справка 🕒 вечер 6

```sh
curl -s $API/orders/<order_id>/certificate -H "Authorization: Bearer $CLIENT" -o cert.pdf
```

### 6.6 Сквозной сценарий целиком 🕒 вечер 6

```sh
scripts/e2e.sh            # успех и дедлайн; агент пишет скрипт, этап 0, п. 9
```

---

## 7. Заглянуть в базы ✅

```sh
docker compose exec users-db     sh -c 'psql -U "$POSTGRES_USER" -d users'
docker compose exec orders-db    sh -c 'psql -U "$POSTGRES_USER" -d orders'
docker compose exec inventory-db sh -c 'psql -U "$POSTGRES_USER" -d inventory'
```

Полезные запросы (после соответствующих вечеров):

```sql
-- Users
SELECT id, name, phone, role FROM users;
SELECT version_id, is_applied FROM goose_db_version ORDER BY id;   -- применённые миграции

-- Orders
SELECT id, status, stage, engineer_id FROM orders ORDER BY created_at DESC LIMIT 5;
SELECT * FROM order_items WHERE order_id = '<id>';
SELECT saga_id, order_id, current_step, deadline FROM saga_state ORDER BY created_at DESC;
SELECT id, event_type, published, created_at FROM outbox ORDER BY created_at DESC LIMIT 10;
SELECT order_id, status, attempts, next_attempt_at FROM certificates;

-- Inventory
SELECT id, name, price, quantity FROM inventory WHERE deleted_at IS NULL;
SELECT saga_id, state FROM stock_attempts ORDER BY created_at DESC;
SELECT id, event_type, published FROM outbox ORDER BY created_at DESC LIMIT 10;
```

Повторно отправить сообщение из outbox (проверка дубля, вечер 5):
```sql
UPDATE outbox SET published = false WHERE id = '<id строки>';
```

---

## 8. RabbitMQ и Gotenberg ✅

```sh
# очереди, сколько сообщений ждёт и сколько консьюмеров подключено
docker compose exec rabbitmq rabbitmqctl list_queues name messages consumers

# exchange и привязки (должны совпадать с definitions в docker-compose.yml)
docker compose exec rabbitmq rabbitmqctl list_exchanges name type
docker compose exec rabbitmq rabbitmqctl list_bindings

# Gotenberg жив?
curl -s http://127.0.0.1:3000/health
```

---

## 9. Сценарии отказов 🕒 вечера 4–7

| Что проверяем | Команды | Чего ждём |
|---|---|---|
| Inventory недоступен при добавлении позиции | `make stop SVC=inventory` → добавить товар | `502 upstream_error`, позиции нет |
| Дедлайн саги | `make stop SVC=inventory` → оплата → подождать 11 с → `make up SVC=inventory` | заказ снова `active`, сага `failed`, в Inventory tombstone `cancelled` |
| Брокер лежит | `make stop SVC=rabbitmq` → оплата → `make up SVC=rabbitmq` | outbox ждёт `published = false`, потом догоняет; сервисы перезапускаются compose |
| Gotenberg лежит | `make stop SVC=gotenberg` → оплата → ждать ~1,5 мин | справка: 3 попытки (10 с, 1 мин), затем `failed` |
| Users лежит | `make stop SVC=users` → вход | `502` в конверте от Gateway |

---

## 10. Сброс и уборка

```sh
make down                        # контейнеры удалены, данные БД остались
docker compose down -v           # ВНИМАНИЕ: удаляет и тома — все данные БД и брокера
make up                          # заново: миграции и seed применятся с нуля
```

Применённые миграции не правятся — ошибка схемы исправляется **новой** миграцией (ADR-024).
Если миграцию правили до того, как её кто-то применил, — проще всего `docker compose down -v`.

---

## 11. Частые проблемы

| Симптом | Причина и что делать |
|---|---|
| `make run`: шелл ругается на `.env` | нет файла `.env` — раздел 2 |
| сервис при старте печатает список переменных | не хватает переменных в `.env` — так и задумано (вечер 1, задача 1) |
| `postgres: ping: … timeout` за 5 с | база не поднята: `make infra`, затем `make ps` |
| `address already in use` | порт занят: другой экземпляр сервиса или чужая программа; `lsof -i :8081` |
| `curl http://sto.localhost` не резолвится | `/etc/hosts`, раздел 2 |
| nginx отвечает `502` на `/api/…` | Gateway не запущен или падает: `make logs SVC=gateway` |
| консьюмер падает на старте «нет очереди» | брокер стартовал без definitions: `make logs SVC=rabbitmq`, проверить `RABBITMQ_USER/PASSWORD` в `.env` |
| `make generate`: ошибка CGO у sqlc | нет gcc — sqlc из Docker (шапка `Makefile`) |
| `make lint` долго в первый раз | собирается golangci-lint; дальше кэш |

---

## 12. Короткий список целей Makefile

| Команда | Что делает |
|---|---|
| `make generate [MODULE=…]` | sqlc + oapi-codegen |
| `make build [MODULE=…]` | `go build ./...` |
| `make test [MODULE=…]` | `go test ./...` |
| `make lint [MODULE=…]` | golangci-lint |
| `make up [SVC="…"]` | собрать и поднять в Docker |
| `make infra` | только БД, RabbitMQ, Gotenberg |
| `make run SVC=…` | сервис на хосте (`gateway users orders inventory`) |
| `make dev SVC=…` | то же, debug-логи через jq |
| `make stop SVC=…` | остановить контейнер(ы) |
| `make logs [SVC=…]` | логи |
| `make ps` | статус контейнеров |
| `make down` | остановить и удалить контейнеры |
