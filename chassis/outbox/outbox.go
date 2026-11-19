// Package outbox persists events in the same transaction as domain changes.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/AlexeBabanov/sto/chassis/rabbit"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// Add uses the transaction supplied by the caller.
func Add(ctx context.Context, tx DBTX, aggregateID uuid.UUID, event string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode outbox payload: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox (id, aggregate_id, event_type, payload) VALUES ($1, $2, $3, $4)`,
		uuid.New(), aggregateID, event, body)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

type message struct {
	ID        uuid.UUID
	EventType string `db:"event_type"`
	Payload   []byte
}

// RunRabbit connects the service's message handler and database outbox to RabbitMQ.
func RunRabbit(ctx context.Context, pool *pgxpool.Pool, brokerURL string, route rabbit.Route, handle rabbit.Handler, logger *slog.Logger) error {
	return rabbit.Run(ctx, brokerURL, route, handle, func(ctx context.Context, publisher *rabbit.Publisher) {
		run(ctx, pool, publisher, logger)
	}, logger)
}

// Each service has one worker. A failed mark can lead to redelivery.
func run(ctx context.Context, pool *pgxpool.Pool, publisher *rabbit.Publisher, logger *slog.Logger) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := publishBatch(ctx, pool, publisher); err != nil && ctx.Err() == nil {
				logger.WarnContext(ctx, "outbox publish failed", "err", err)
			}
		}
	}
}

func publishBatch(ctx context.Context, pool *pgxpool.Pool, publisher *rabbit.Publisher) error {
	rows, err := pool.Query(ctx, `SELECT id, event_type, payload FROM outbox WHERE NOT published ORDER BY created_at, id LIMIT 20`)
	if err != nil {
		return err
	}
	pending, err := pgx.CollectRows(rows, pgx.RowToStructByName[message])
	if err != nil {
		return err
	}
	for _, msg := range pending {
		if err := publisher.Publish(ctx, msg.ID.String(), msg.EventType, msg.Payload); err != nil {
			return fmt.Errorf("publish %s: %w", msg.ID, err)
		}
		if _, err := pool.Exec(ctx, `UPDATE outbox SET published = true, published_at = now() WHERE id = $1 AND NOT published`, msg.ID); err != nil {
			return fmt.Errorf("mark %s published: %w", msg.ID, err)
		}
	}
	return nil
}
