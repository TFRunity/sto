package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InTx runs work in a transaction with PostgreSQL's default isolation level.
func InTx(ctx context.Context, pool *pgxpool.Pool, work func(pgx.Tx) error) error {
	return inTx(ctx, pool, pgx.TxOptions{}, work)
}

// InReadSnapshot keeps multiple reads on the same PostgreSQL snapshot.
func InReadSnapshot(ctx context.Context, pool *pgxpool.Pool, work func(pgx.Tx) error) error {
	return inTx(ctx, pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, work)
}

func inTx(ctx context.Context, pool *pgxpool.Pool, options pgx.TxOptions, work func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := work(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
