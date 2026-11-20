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

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load[Config]()
	if err != nil {
		return fmt.Errorf("load users config: %w", err)
	}

	logger := logx.New("users", cfg.LogLevel)
	logger.Info("users started",
		"addr", cfg.HTTPAddr,
		"log_level", cfg.LogLevel,
		"jwt_ttl", cfg.JWTTTL.String(),
	)

	pool, err := postgres.ConnectAndMigrate(ctx, cfg.DatabaseURL.Reveal(), migrations.FS, logger)
	if err != nil {
		return fmt.Errorf("prepare users database: %w", err)
	}
	defer pool.Close()

	repo := internal.NewRepository(pool)
	svc, err := internal.NewService(repo, []byte(cfg.JWTSecret.Reveal()), cfg.JWTTTL)
	if err != nil {
		return fmt.Errorf("initialize users service: %w", err)
	}

	app, err := newApp(svc, logger)
	if err != nil {
		return fmt.Errorf("initialize users HTTP application: %w", err)
	}

	if err := httpx.Run(ctx, cfg.HTTPAddr, app, logger); err != nil {
		return fmt.Errorf("run users HTTP server: %w", err)
	}
	logger.Info("users stopped")
	return nil
}

// newApp — валидатор по спецификации → strict-сервер → наш Handler (ADR-041).
// Валидатор снаружи: иначе strict-сервер молча проглотил бы лишние поля.
func newApp(svc *internal.Service, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, fmt.Errorf("load openapi spec: %w", err)
	}
	handleError := httpx.StrictErrorHandler(logger)
	strict := api.NewStrictHandlerWithOptions(internal.NewHandler(svc), nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpx.StrictRequestErrorHandler(logger),
		ResponseErrorHandlerFunc: handleError,
	})
	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}
