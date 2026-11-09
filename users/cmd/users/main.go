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
	"github.com/AlexeBabanov/sto/users/internal"
	"github.com/AlexeBabanov/sto/users/internal/api"
	"github.com/AlexeBabanov/sto/users/migrations"
)

// Порядок: сначала перехват сигналов, потом конфиг — так принято.
// Поток данных при старте — users/uml/users-flow.drawio, страница 1.
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
	logger.Info("migrations done", "applied", applied)

	repo := internal.NewRepository(pool)
	svc, err := internal.NewService(repo, []byte(cfg.JWTSecret.Reveal()), cfg.JWTTTL)
	if err != nil {
		logger.Error("service init failed", "err", err)
		os.Exit(1)
	}

	app, err := newApp(svc, logger)
	if err != nil {
		logger.Error("app init failed", "err", err)
		os.Exit(1)
	}

	if err := httpx.Run(ctx, cfg.HTTPAddr, app, logger); err != nil {
		logger.Error("http server failed", "err", err)
		os.Exit(1)
	}
	logger.Info("users stopped")
}

// newApp — валидатор по спецификации → strict-сервер → наш Handler (ADR-041).
// Валидатор снаружи: иначе strict-сервер молча проглотил бы лишние поля.
func newApp(svc *internal.Service, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	strict := api.NewStrictHandlerWithOptions(internal.NewHandler(svc), nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpx.StrictErrorHandler(logger),
		ResponseErrorHandlerFunc: httpx.StrictErrorHandler(logger),
	})
	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}
