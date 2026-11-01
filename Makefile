# Корневой Makefile монорепо (ADR-014, ADR-036, I-08).
# Те же цели вызывает CI — локальная проверка и CI не расходятся.
#
# Один модуль:  make test MODULE=orders
# Все модули:   make test

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

.PHONY: generate build test lint up down

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

up:
	docker compose up -d --build

down:
	docker compose down
