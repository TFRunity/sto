// Package config — общая часть загрузки конфигурации сервисов (ADR-042).
package config

// Пустой импорт держит зависимость в go.mod, пока нет кода (go mod tidy её не
// удалит). Когда появится вызов env.ParseAs, заменить на обычный импорт.
import _ "github.com/caarlos0/env/v11"

// TODO(dev): общая обёртка над caarlos0/env.                          → ADR-042
//   - разбор структуры конфига сервиса (у каждого сервиса своя, с тегами
//     env:"NAME,required" / envDefault) — все ошибки сразу, при старте
//   - LOG_LEVEL → slog.Level
//   - секреты (JWT_SECRET, пароли в DATABASE_URL, SEED_*) не логируются → ADR-011
// Имена переменных — в .env.example и docker-compose.yml.
