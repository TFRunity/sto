// Package postgres — подключение к PostgreSQL и миграции при старте (ADR-024).
package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// pingTimeout — сколько ждать ответа базы при старте. Без него недоступная
// база (молчащий порт) вешает сервис навсегда.
const pingTimeout = 5 * time.Second

// Connect открывает пул соединений и проверяет, что база отвечает.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("postgres: bad config: %w", err)
	}
	if err := ping(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}

// ping проверяет, что база отвечает, но ждёт не дольше pingTimeout.
// Таймаут строится от ctx: Ctrl+C во время ожидания прерывает его сразу.
func ping(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	return pool.Ping(ctx)
}

func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) (int, error) {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

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
