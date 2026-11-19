package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexeBabanov/sto/chassis/config"
	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/logx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/inventory/internal/api"
	"github.com/AlexeBabanov/sto/inventory/internal/item"
	"github.com/AlexeBabanov/sto/inventory/migrations"
)

type Config struct {
	config.Base

	DatabaseURL config.Secret `env:"DATABASE_URL,notEmpty"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load[Config]()
	if err != nil {
		return fmt.Errorf("load inventory config: %w", err)
	}
	logger := logx.New("inventory", cfg.LogLevel)

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL.Reveal())
	if err != nil {
		return fmt.Errorf("connect to inventory database: %w", err)
	}
	defer pool.Close()

	applied, err := postgres.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return fmt.Errorf("apply inventory migrations: %w", err)
	}
	logger.Info("migrations done", "applied", applied)

	handler := item.NewHandler(item.NewRepository(pool))
	app, err := newApp(handler, logger)
	if err != nil {
		return err
	}

	if err := httpx.Run(ctx, cfg.HTTPAddr, app, logger); err != nil {
		return fmt.Errorf("run inventory HTTP server: %w", err)
	}
	return nil
}

func newApp(handler *item.Handler, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load inventory OpenAPI spec: %w", err)
	}

	handleError := httpx.StrictErrorHandler(logger)
	strict := api.NewStrictHandlerWithOptions(handler, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handleError,
		ResponseErrorHandlerFunc: handleError,
	})

	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}
