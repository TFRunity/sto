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

MODULES  := chassis gateway users orders vehicle inventory services
MODULE   ?=
TARGETS  := $(if $(MODULE),$(MODULE),$(MODULES))

# Инструменты запускаются через `go run pkg@version`: версия зафиксирована здесь,
# глобальная установка не нужна, golangci-lint собирается текущим Go.
# Любой можно переопределить: make generate SQLC=sqlc
# sqlc собирается с CGO (pg_query_go) — нужен gcc. Без него:
#   SQLC='docker run --rm -v $(CURDIR):/src -w /src/$$m sqlc/sqlc:1.29.0'
SQLC         ?= go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
OAPI_CODEGEN ?= go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.0
GOLANGCI     ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0

SVC   ?=
INFRA := users-db orders-db inventory-db rabbitmq gotenberg

.PHONY: generate build test lint up infra stop logs ps down

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

down:
	docker compose down
