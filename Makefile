# Корневой Makefile монорепо (ADR-014, ADR-036, I-08).
# Те же цели вызывает CI — локальная проверка и CI не расходятся.
#
# Один модуль:  make test MODULE=orders
# Все модули:   make test
#
# Окружение (SVC — сервисы docker-compose.yml, можно несколько):
#   make up                       всё
#   make up SVC=users             users + то, от чего он зависит (users-db)
#   make up SVC="users gateway"
#   make infra                    только БД, RabbitMQ, Gotenberg
#   make stop SVC=inventory       остановить, не удаляя (сценарий дедлайна)
#   make logs SVC=orders · make ps · make down
#
# Сервис на хосте (ADR-040): сначала make infra, потом
#   make run SVC=users            go run с переменными из .env и хостовыми адресами
#   make dev SVC=users            то же, но LOG_LEVEL=debug и логи через jq

MODULES  := chassis gateway users orders vehicle inventory services
MODULE   ?=
TARGETS  := $(if $(MODULE),$(MODULE),$(MODULES))

# Инструменты запускаются через `go run pkg@version`: версия зафиксирована здесь,
# глобальная установка не нужна, golangci-lint собирается текущим Go.
# Любой можно переопределить: make generate SQLC=sqlc
# sqlc собирается с CGO (pg_query_go) — нужен gcc. Нет gcc — sqlc из Docker:
#   make generate SQLC='docker run --rm -v $(CURDIR):/src -w /src/$$m sqlc/sqlc:1.29.0'
#   ($$m — каталог модуля из цикла в цели generate; Docker запускается в нём)
SQLC         ?= go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
OAPI_CODEGEN ?= go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0
GOLANGCI     ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

SVC   ?=
INFRA := users-db orders-db inventory-db rabbitmq gotenberg

# Хостовые адреса для make run (ADR-040). Остальное (JWT_SECRET, LOG_LEVEL, …) — из .env.
# Сервис слушает только 127.0.0.1; соседи — тоже на хосте.
# $$VAR — это $VAR для шелла (одиночный $ съедает Make); раскрывается после чтения .env.
RUN_SERVICES := gateway users orders inventory
AMQP_URL := amqp://$$RABBITMQ_USER:$$RABBITMQ_PASSWORD@127.0.0.1:5672/

RUN_ENV_gateway   := HTTP_ADDR=127.0.0.1:8080 \
	USERS_URL=http://127.0.0.1:8081 ORDERS_URL=http://127.0.0.1:8082 INVENTORY_URL=http://127.0.0.1:8083
RUN_ENV_users     := HTTP_ADDR=127.0.0.1:8081 \
	DATABASE_URL="postgres://$$USERS_DB_USER:$$USERS_DB_PASSWORD@127.0.0.1:5433/users?sslmode=disable"
RUN_ENV_orders    := HTTP_ADDR=127.0.0.1:8082 \
	DATABASE_URL="postgres://$$ORDERS_DB_USER:$$ORDERS_DB_PASSWORD@127.0.0.1:5434/orders?sslmode=disable" \
	RABBITMQ_URL="$(AMQP_URL)" \
	INVENTORY_URL=http://127.0.0.1:8083 USERS_URL=http://127.0.0.1:8081 GOTENBERG_URL=http://127.0.0.1:3000
RUN_ENV_inventory := HTTP_ADDR=127.0.0.1:8083 \
	DATABASE_URL="postgres://$$INVENTORY_DB_USER:$$INVENTORY_DB_PASSWORD@127.0.0.1:5435/inventory?sslmode=disable" \
	RABBITMQ_URL="$(AMQP_URL)"

.PHONY: generate build test lint up infra stop logs ps down run dev

# sqlc — модули с sqlc.yaml; oapi-codegen — модули с oapi-codegen.yaml
# (конфиг генерации по своей части api/openapi.yaml, ADR-028).
generate:
	@set -e; for m in $(TARGETS); do \
		if [ -f $$m/sqlc.yaml ]; then \
			echo "sqlc: $$m"; (cd $$m && $(SQLC) generate); \
		fi; \
		if [ -f $$m/oapi-codegen.yaml ]; then \
			echo "oapi-codegen: $$m"; (cd $$m && $(OAPI_CODEGEN) -config oapi-codegen.yaml ../api/openapi.yaml); \
		fi; \
	done

build:
	@set -e; for m in $(TARGETS); do \
		echo "build: $$m"; (cd $$m && go build ./...); \
	done

test:
	@set -e; for m in $(TARGETS); do \
		echo "test: $$m"; (cd $$m && go test ./...); \
	done

# Конфиг .golangci.yml в корне — golangci-lint находит его, поднимаясь по каталогам.
lint:
	@set -e; for m in $(TARGETS); do \
		echo "lint: $$m"; (cd $$m && $(GOLANGCI) run ./...); \
	done

# Зависимости из depends_on compose поднимает сам.
up:
	docker compose up -d --build $(SVC)

infra:
	docker compose up -d $(INFRA)

stop:
	docker compose stop $(SVC)

logs:
	docker compose logs -f --tail=100 $(SVC)

ps:
	docker compose ps -a

# RUN_ENV_$(SVC) — переменные выбранного сервиса: SVC=users → RUN_ENV_users.
run:
ifeq ($(filter $(SVC),$(RUN_SERVICES)),)
	$(error укажи сервис: make run SVC=<$(RUN_SERVICES)>)
endif
	set -a; . ./.env; set +a; cd $(SVC) && env $(RUN_ENV_$(SVC)) go run ./cmd/$(SVC)

# То же, что run, но для чтения глазами: LOG_LEVEL=debug и вывод через jq (нужен в системе).
# Фильтр jq: строка-JSON разворачивается, любая другая печатается как есть.
dev:
ifeq ($(filter $(SVC),$(RUN_SERVICES)),)
	$(error укажи сервис: make dev SVC=<$(RUN_SERVICES)>)
endif
	@set -a; . ./.env; set +a; cd $(SVC) && env $(RUN_ENV_$(SVC)) LOG_LEVEL=debug go run ./cmd/$(SVC) \
		| jq -R '. as $$line | try fromjson catch $$line'

down:
	docker compose down

dev-up: infra
	@echo "==> Инфраструктура проверена/запущена."
	@echo "==> Запускаем $(SVC) в отладчике (dlv headless на :2345)..."
	@set -a; . ./.env; set +a; cd $(SVC) && env $(RUN_ENV_$(SVC)) LOG_LEVEL=debug \
		dlv debug ./cmd/$(SVC) --headless --listen=:2345 --api-version=2 --accept-multiclient --check-go-version=false
