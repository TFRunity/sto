// Package config — загрузка конфигурации сервиса из окружения (ADR-026, P-12).
package config

// TODO(dev): загрузка конфига из env.
//   - os.LookupEnv + своя структура, без библиотек                  → ADR-026
//   - все обязательные переменные проверяются при старте,
//     ошибки собираются через errors.Join, а не по первой            → ADR-026, I-04
//   - разбор типов в одном месте (time.ParseDuration, strconv)      → ADR-026
//   - секреты (JWT_SECRET, пароли в DATABASE_URL) не логируются     → ADR-011
//   - уровень логов slog (LOG_LEVEL) — отсюда же
// Имена переменных — в .env.example и docker-compose.yml.
