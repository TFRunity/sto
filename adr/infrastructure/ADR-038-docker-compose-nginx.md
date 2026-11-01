# ADR-038: Docker, Compose, PostgreSQL 18, nginx — локальное окружение v1

## Статус
Принято

## Контекст
Все компоненты v1 запускаются в Docker Compose (ADR-010–012). Нужно
зафиксировать образы, сети, порядок запуска, версии и роль nginx.
Публичного развёртывания пока нет — только локальный запуск.

## Решение
**Образы сервисов (I-01):** multi-stage сборка (`CGO_ENABLED=0`);
runtime — `alpine` (тег зафиксирован), в образе непривилегированный
пользователь; healthcheck через `wget` на `/healthz`; **отдельный
Dockerfile у каждого из шести сервисов**; контекст сборки — корень
репозитория (ADR-036).

**Compose (I-02):**
- пять контейнеров PostgreSQL — по одному на сервис с БД (ADR-001);
- сети: `edge` (nginx ↔ Gateway), `backend` (Gateway ↔ сервисы ↔
  RabbitMQ ↔ Gotenberg), отдельная сеть на каждую БД (сервис ↔ его
  Postgres);
- `depends_on` с `condition: service_healthy` на свой Postgres
  (`pg_isready`) и RabbitMQ;
- `restart: unless-stopped` у всех сервисов (сервис падает при
  отсутствии своих очередей — ADR-025);
- `ports:` только у nginx (и Gateway, если нужно в разработке).

**PostgreSQL (I-03):** образ `postgres:18` (тег мажорной версии).
⚠ В образе 18+ VOLUME — `/var/lib/postgresql` (данные в
`/var/lib/postgresql/18/docker`); монтирование в старый путь
`/var/lib/postgresql/data` ломает запуск. Переход на 19 — после выхода
(GA запланирован на 19.11.2026) и обкатки.

**`.env` (I-04):** `.env` в `.gitignore` с первого коммита; в git —
`.env.example` со всеми переменными и фиктивными значениями.

**RabbitMQ (I-05):** definitions встроенным текстом в compose
(`configs:` → `content:`) с подстановкой из `.env` — набросок (сверить
формат с документацией RabbitMQ при реализации):
```yaml
configs:
  rabbit_definitions:
    content: |
      {"users": [{"name": "${RABBITMQ_USER}", "password": "${RABBITMQ_PASSWORD}", "tags": "administrator"}],
       "vhosts": [{"name": "/"}],
       "permissions": [{"user": "${RABBITMQ_USER}", "vhost": "/", "configure": ".*", "write": ".*", "read": ".*"}],
       "exchanges": […], "queues": […], "bindings": […]}
```

**Gotenberg:** см. ADR-027.

**nginx (I-06, P-15):**
- единственная точка входа; один origin: статика веба (`/`) и API
  (`/api/` → Gateway) — поэтому **CORS не нужен** ни в nginx, ни в
  Gateway; в разработке dev-сервер фронта проксирует `/api`;
- пока только HTTP; веб — по имени `sto.localhost` (браузеры резолвят
  `*.localhost` в 127.0.0.1 без правки `hosts`); nginx слушает не только
  127.0.0.1;
- RN на телефоне — по IP машины в локальной сети, Android-эмулятор —
  `10.0.2.2`; адрес API в RN задаётся конфигом; для разработки в RN явно
  разрешается открытый HTTP (Android cleartext, iOS ATS);
- `client_max_body_size 4m` (ADR-029).

**`/healthz` (I-09):** только «процесс жив», всегда 200. Compose не
перезапускает unhealthy-контейнер; healthcheck нужен для порядка запуска
и наглядности. Готовность БД гарантирует healthcheck самого Postgres.

## Альтернативы, которые рассматривались
- **`distroless/static:nonroot` + подкоманда healthcheck в бинаре** —
  меньше поверхность атаки, но нет шелла для отладки. Отклонено.
- **Один Dockerfile с `ARG SERVICE`** — меньше копий, но сложнее
  настраивать отдельный сервис. Отклонено.
- **Один Postgres с пятью базами** — изоляция правами, а не сетью.
  Отклонено.
- **`/healthz` с ping БД** — отклонено ради простоты.

## Последствия
**Плюсы:** ADR-001 обеспечен сетью; порядок запуска детерминирован;
секреты только в `.env`.

**Минусы:** пять томов и пять бэкапов; общие изменения сборки вносятся в
шесть Dockerfile; JSON топологии RabbitMQ внутри compose-файла.

**Долг:** HTTPS (TLS на nginx) обязателен при публичном развёртывании —
иначе JWT уходит открытым текстом.

## Дата
2026-10-30
