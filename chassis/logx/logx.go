package logx

import (
	"log/slog"
	"os"
)

// New создаёт логгер сервиса с минимальным уровнем level.
func New(service string, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", service)
}
