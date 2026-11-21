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
	"github.com/AlexeBabanov/sto/gateway/internal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load[Config]()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logger := logx.New("gateway", cfg.LogLevel)
	logger.Info("gateway started",
		"addr", cfg.HTTPAddr,
		"log_level", cfg.LogLevel,
		"users_url", cfg.UsersURL.String(),
		"orders_url", cfg.OrdersURL.String(),
		"inventory_url", cfg.InventoryURL.String(),
	)
	app, err := newApp(cfg, logger)
	if err != nil {
		logger.Error("initialize director routes", "err", err)
		os.Exit(1)
	}
	if err := httpx.Run(ctx, cfg.HTTPAddr, app, logger); err != nil {
		logger.Error("http server failed", "err", err)
		os.Exit(1)
	}
	logger.Info("gateway stopped")
}

// newApp wires both the proxied services and Gateway's own director routes.
func newApp(cfg Config, logger *slog.Logger) (http.Handler, error) {
	director, err := internal.NewDirector(cfg.OrdersURL, cfg.UsersURL, cfg.VehicleURL, logger)
	if err != nil {
		return nil, err
	}
	proxies := map[string]http.Handler{
		"users":     internal.NewProxy(&cfg.UsersURL, logger),
		"orders":    internal.NewProxy(&cfg.OrdersURL, logger),
		"inventory": internal.NewProxy(&cfg.InventoryURL, logger),
		"director":  director,
	}
	if cfg.VehicleURL.Host != "" {
		proxies["vehicle"] = internal.NewProxy(&cfg.VehicleURL, logger)
	}
	if cfg.ServicesURL.Host != "" {
		proxies["services"] = internal.NewProxy(&cfg.ServicesURL, logger)
	}
	return internal.NewRouter([]byte(cfg.JWTSecret), proxies, logger), nil
}
