// Package postgres — подключение к PostgreSQL и миграции при старте (ADR-024).
package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// pingTimeout — сколько ждать ответа базы при старте. Без него недоступная
// база (молчащий порт) вешает сервис навсегда.
const pingTimeout = 5 * time.Second

// Connect открывает пул соединений и проверяет, что база отвечает.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("postgres: bad config: %w", err)
	}

	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	if err := ping(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}

// ConnectAndMigrate возвращает готовый пул. При ошибке миграции закрывает его.
func ConnectAndMigrate(ctx context.Context, url string, migrations fs.FS, logger *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := Connect(ctx, url)
	if err != nil {
		return nil, err
	}
	applied, err := Migrate(ctx, pool, migrations)
	if err != nil {
		pool.Close()
		return nil, err
	}
	logger.InfoContext(ctx, "migrations done", "applied", applied)
	return pool, nil
}

// ping проверяет, что база отвечает, но ждёт не дольше pingTimeout.
// Таймаут строится от ctx: Ctrl+C во время ожидания прерывает его сразу.
func ping(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	return pool.Ping(ctx)
}

// Migrate применяет миграции сервиса из переданной файловой системы через goose.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) (int, error) {
	db := stdlib.OpenDBFromPool(pool)
	// Ошибка закрытия не нужна: db — обёртка над пулом на время миграций.
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		return 0, fmt.Errorf("postgres: load migrations: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: migrate up: %w", err)
	}
	return len(results), err
}
