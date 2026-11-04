// Package config — общая часть загрузки конфигурации сервисов (ADR-042).
package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
)

// Прописывать в каждом конфиге сервиса
type Base struct {
	HTTPAddr string     `env:"HTTP_ADDR,required"`
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"info"`
}

// Load читает конфиг сервиса T из окружения процесса.
// Ошибка возвращается в виде env.AggregateError
func Load[T any]() (T, error) {
	return load[T](nil)
}

// load - то же с явным окружением;
// nil - окружение процесса. Нужна тестам.
func load[T any](environ map[string]string) (T, error) {
	return env.ParseAsWithOptions[T](env.Options{Environment: environ})
}
