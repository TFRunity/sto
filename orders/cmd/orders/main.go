package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	pdf "github.com/AlexeBabanov/sto/chassis/certificate"
	"github.com/AlexeBabanov/sto/chassis/config"
	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/logx"
	"github.com/AlexeBabanov/sto/chassis/outbox"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/rabbit"
	"github.com/AlexeBabanov/sto/orders/internal/certificate"
	"github.com/AlexeBabanov/sto/orders/internal/order"
	"github.com/AlexeBabanov/sto/orders/internal/saga"
	"github.com/AlexeBabanov/sto/orders/migrations"
	"golang.org/x/sync/errgroup"
)

type ordersConfig struct {
	config.Base
	RabbitMQURL  config.Secret `env:"RABBITMQ_URL,notEmpty"`
	DatabaseURL  config.Secret `env:"DATABASE_URL,notEmpty"`
	GotenbergURL url.URL       `env:"GOTENBERG_URL,notEmpty"`
	InventoryURL url.URL       `env:"INVENTORY_URL,notEmpty"`
	VehicleURL   url.URL       `env:"VEHICLE_URL"`
	ServicesURL  url.URL       `env:"SERVICES_URL"`
	UsersURL     url.URL       `env:"USERS_URL,notEmpty"`
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

	pool, err := postgres.ConnectAndMigrate(ctx, cfg.DatabaseURL.Reveal(), migrations.FS, logger)
	if err != nil {
		return fmt.Errorf("prepare orders database: %w", err)
	}
	defer pool.Close()

	payments := saga.New(pool)
	certificates := certificate.NewWorker(pool, order.NewCertificateSource(pool, cfg.InventoryURL, cfg.UsersURL), pdf.NewClient(cfg.GotenbergURL), logger)
	app, err := order.NewHTTPHandler(pool, cfg.InventoryURL, cfg.VehicleURL, cfg.ServicesURL, payments, logger)
	if err != nil {
		return fmt.Errorf("load orders OpenAPI spec: %w", err)
	}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error { return httpx.Run(ctx, cfg.HTTPAddr, app, logger) })
	group.Go(func() error {
		route := rabbit.Route{Exchange: "saga_commands", RoutingKey: "inventory.saga_commands", Queue: "orders.saga_replies"}
		handle := func(ctx context.Context, event string, body []byte) error {
			return payments.HandleReply(ctx, event, body, logger)
		}
		return outbox.RunRabbit(ctx, pool, cfg.RabbitMQURL.Reveal(), route, handle, logger)
	})
	group.Go(func() error { payments.RunDeadlines(ctx, logger); return nil })
	group.Go(func() error { certificates.Run(ctx); return nil })
	return group.Wait()
}
