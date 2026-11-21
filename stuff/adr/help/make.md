# Make в проекте

Запускай команды из корня репозитория. В примере `make test MODULE=users`
`test` — имя цели, а `MODULE=users` — значение переменной для этого запуска.
Рецепт цели лежит в [Makefile](../../Makefile) под строкой `test:`.

| Команда | Действие |
| --- | --- |
| `make generate MODULE="orders inventory"` | Генерирует Go из SQL и OpenAPI для двух модулей. См. [инструкцию](generate.md). |
| `make build MODULE=users` | Выполняет `go build ./...` в Users. |
| `make test MODULE=users` | Выполняет `go test ./...` в Users. |
| `make lint MODULE=users` | Запускает golangci-lint в Users с корневым `.golangci.yml`. |
| `make lint test` | Запускает линт и тесты по всем модулям. |
| `make infra` | Поднимает БД, RabbitMQ и Gotenberg в Compose. |
| `make up SVC="gateway users"` | Собирает образы и запускает сервисы с зависимостями Compose. |
| `make ps` | Показывает состояние контейнеров Compose. |
| `make logs SVC=users` | Показывает последние 100 строк лога Users и следит за новыми. |
| `make stop SVC=users` | Останавливает контейнер Users. Без `SVC` останавливает сервисы проекта. |
| `make down` | Останавливает и удаляет контейнеры и сети Compose, сохраняя именованные тома. |
| `make run SVC=users` | Запускает один сервис на хосте с настройками из `.env` и адресами из Makefile. |
| `make dev SVC=users` | Запускает сервис на хосте с debug-логами, пропущенными через `jq`. |
| `make dev-up SVC=users` | Поднимает инфраструктуру и запускает Users через Delve. См. [подключение и точки останова](delve.md). |

`MODULE` выбирает каталоги для генерации, сборки, тестов и линта.
`SVC` выбирает сервис для Docker Compose или запуска на хосте. Примеры полного
запуска и проверки через curl есть в [RUNNING.md](../../RUNNING.md).

Для просмотра раскрытого рецепта используй `make -n generate MODULE=orders`.
Для этой цели Make напечатает shell-команду с раскрытыми переменными без запуска
генераторов. Если запуск завершился ошибкой, начни с сообщения программы перед
строкой `make: ***`, затем проверь рецепт и рабочий каталог этой программы.
