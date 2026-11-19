package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexeBabanov/sto/chassis/config"
	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/logx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/orders/internal/order"
	"github.com/AlexeBabanov/sto/orders/migrations"
)

type ordersConfig struct {
	config.Base
	DatabaseURL  config.Secret `env:"DATABASE_URL,notEmpty"`
	InventoryURL url.URL       `env:"INVENTORY_URL,notEmpty"`
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
	cfg, err := config.Load[ordersConfig]()
	if err != nil {
		return fmt.Errorf("load orders config: %w", err)
	}
	logger := logx.New("orders", cfg.LogLevel)

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL.Reveal())
	if err != nil {
		return fmt.Errorf("connect to orders database: %w", err)
	}
	defer pool.Close()

	applied, err := postgres.Migrate(ctx, pool, migrations.FS)
	if err != nil {
		return fmt.Errorf("apply orders migrations: %w", err)
	}
	logger.Info("migrations done", "applied", applied)

	app, err := order.NewHTTPHandler(pool, cfg.InventoryURL, logger)
	if err != nil {
		return fmt.Errorf("load orders OpenAPI spec: %w", err)
	}
	return httpx.Run(ctx, cfg.HTTPAddr, app, logger)
}
