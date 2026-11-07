package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexeBabanov/sto/chassis/config"
	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/logx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/users/migrations"
)

// Сначала нужно делать первые 2 строчки, и только потом загружать
// конфиг - так принято

// TODO(dev): сборка Users.
//   - pgxpool; миграции goose при старте (go:embed)                  → ADR-024
//   - репозиторий → сервис → хендлеры; middleware идентичности
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load[Config]()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger := logx.New("users", cfg.LogLevel)
	logger.Info("users started",
		"addr", cfg.HTTPAddr,
		"log_level", cfg.LogLevel,
		"jwt_ttl", cfg.JWTTTL.String(),
	)

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL.Reveal())
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("db connected")

	applied, err := postgres.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		logger.Error("migrations failed", "err", err)
		os.Exit(1)
	}
	logger.Info("Migrations done", "applied", applied)

	if err := httpx.Run(ctx, cfg.HTTPAddr, http.NotFoundHandler(), logger); err != nil {
		logger.Error("http server failed", "err", err)
		os.Exit(1)
	}
	logger.Info("users stopped")
}
