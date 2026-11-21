package main

import (
	"context"
	"fmt"
	"net/http"
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
	PhotoDir    string        `env:"PHOTO_DIR,required,notEmpty"`
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
	options := httpx.NewStrictErrors(logger)
	strict := api.NewStrictHandlerWithOptions(vehicle.NewHandler(pool, cfg.PhotoDir), nil, api.StrictHTTPServerOptions(options))
	routes := api.Handler(strict)
	mux := http.NewServeMux()
	// Orders вызывает этот маршрут внутри Compose. Gateway его не публикует.
	mux.Handle("GET /internal/vehicles/{id}/ownership", routes)
	mux.Handle("GET /internal/vehicles/search", routes)
	mux.Handle("GET /internal/vehicles", routes)
	mux.Handle("/", httpx.RequireIdentity(routes, logger))
	app := httpx.Validator(spec, logger)(mux)
	return httpx.Run(ctx, cfg.HTTPAddr, vehicle.LimitPhotoUpload(app, options.ResponseErrorHandlerFunc), logger)
}
