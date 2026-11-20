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
	"github.com/AlexeBabanov/sto/chassis/outbox"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/rabbit"
	"github.com/AlexeBabanov/sto/inventory/internal/api"
	"github.com/AlexeBabanov/sto/inventory/internal/item"
	"github.com/AlexeBabanov/sto/inventory/internal/stock"
	"github.com/AlexeBabanov/sto/inventory/migrations"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	config.Base
	RabbitMQURL config.Secret `env:"RABBITMQ_URL,notEmpty"`

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

	pool, err := postgres.ConnectAndMigrate(ctx, cfg.DatabaseURL.Reveal(), migrations.FS, logger)
	if err != nil {
		return fmt.Errorf("prepare inventory database: %w", err)
	}
	defer pool.Close()

	handler := item.NewHandler(item.NewRepository(pool))
	app, err := newApp(handler, logger)
	if err != nil {
		return err
	}

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return httpx.Run(ctx, cfg.HTTPAddr, app, logger) })
	group.Go(func() error {
		route := rabbit.Route{Exchange: "saga_replies", RoutingKey: "orders.saga_replies", Queue: "inventory.saga_commands"}

		return outbox.RunRabbit(ctx, pool, cfg.RabbitMQURL.Reveal(), route, stock.NewHandler(pool), logger)
	})
	return group.Wait()
}

func newApp(handler *item.Handler, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load inventory OpenAPI spec: %w", err)
	}

	handleError := httpx.StrictErrorHandler(logger)
	strict := api.NewStrictHandlerWithOptions(handler, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpx.StrictRequestErrorHandler(logger),
		ResponseErrorHandlerFunc: handleError,
	})

	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}
