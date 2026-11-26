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
	"github.com/AlexeBabanov/sto/chat/internal/dialog"
	"github.com/AlexeBabanov/sto/chat/migrations"
	"github.com/google/uuid"
)

type Config struct {
	config.Base

	DatabaseURL config.Secret `env:"DATABASE_URL,notEmpty"`
	OrdersURL   url.URL       `env:"ORDERS_URL,notEmpty"`
	UsersURL    url.URL       `env:"USERS_URL,notEmpty"`
	SupportID   string        `env:"CHAT_SUPPORT_DIRECTOR_ID,notEmpty"`
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
		return fmt.Errorf("load chat config: %w", err)
	}
	supportID, err := uuid.Parse(cfg.SupportID)
	if err != nil || supportID == uuid.Nil {
		return fmt.Errorf("CHAT_SUPPORT_DIRECTOR_ID must be a non-zero UUID")
	}
	logger := logx.New("chat", cfg.LogLevel)
	pool, err := postgres.ConnectAndMigrate(ctx, cfg.DatabaseURL.Reveal(), migrations.FS, logger)
	if err != nil {
		return fmt.Errorf("prepare chat database: %w", err)
	}
	defer pool.Close()

	app, err := dialog.NewHTTPHandler(pool, cfg.OrdersURL, cfg.UsersURL, supportID, logger)
	if err != nil {
		return fmt.Errorf("load chat OpenAPI spec: %w", err)
	}
	return httpx.Run(ctx, cfg.HTTPAddr, app, logger)
}
