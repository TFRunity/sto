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
	"github.com/AlexeBabanov/sto/services/internal"
	"github.com/AlexeBabanov/sto/services/internal/api"
	"github.com/AlexeBabanov/sto/services/migrations"
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
		return fmt.Errorf("load services config: %w", err)
	}
	logger := logx.New("services", cfg.LogLevel)

	pool, err := postgres.ConnectAndMigrate(ctx, cfg.DatabaseURL.Reveal(), migrations.FS, logger)
	if err != nil {
		return fmt.Errorf("prepare services database: %w", err)
	}
	defer pool.Close()

	handler := internal.NewHandler(pool)
	app, err := newApp(handler, logger)
	if err != nil {
		return err
	}

	return httpx.Run(ctx, cfg.HTTPAddr, app, logger)
}

func newApp(handler *internal.Handler, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load services OpenAPI spec: %w", err)
	}

	strict := api.NewStrictHandlerWithOptions(handler, nil, api.StrictHTTPServerOptions(httpx.NewStrictErrors(logger)))

	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}
