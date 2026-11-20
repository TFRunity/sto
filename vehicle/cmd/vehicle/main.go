package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexeBabanov/sto/chassis/config"
	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/logx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	vehicle "github.com/AlexeBabanov/sto/vehicle/internal"
	"github.com/AlexeBabanov/sto/vehicle/internal/api"
	"github.com/AlexeBabanov/sto/vehicle/migrations"
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
		return fmt.Errorf("load vehicle config: %w", err)
	}
	logger := logx.New("vehicle", cfg.LogLevel)

	pool, err := postgres.ConnectAndMigrate(ctx, cfg.DatabaseURL.Reveal(), migrations.FS, logger)
	if err != nil {
		return fmt.Errorf("prepare vehicle database: %w", err)
	}
	defer pool.Close()

	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("load vehicle OpenAPI spec: %w", err)
	}
	handleError := httpx.StrictErrorHandler(logger)
	strict := api.NewStrictHandlerWithOptions(vehicle.NewHandler(pool), []api.StrictMiddlewareFunc{vehicle.ReadIdentity}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handleError,
		ResponseErrorHandlerFunc: handleError,
	})
	app := httpx.Validator(spec, logger)(api.Handler(strict))
	return httpx.Run(ctx, cfg.HTTPAddr, app, logger)
}
