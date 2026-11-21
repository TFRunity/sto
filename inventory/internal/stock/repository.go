package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlexeBabanov/sto/chassis/outbox"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/stockmessage"
	"github.com/AlexeBabanov/sto/inventory/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	pool *pgxpool.Pool
}

func (r repository) apply(ctx context.Context, event string, command stockmessage.Deduction) error {
	return postgres.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		state, err := lockAttempt(ctx, q, event, command.Attempt)
		if err != nil {
			return err
		}
		return applyCommand(ctx, tx, q, decide(state, event), command)
	})
}

func lockAttempt(ctx context.Context, q *db.Queries, event string, attempt stockmessage.Attempt) (string, error) {
	initial := rejected
	if event == stockmessage.CancelStock {
		initial = cancelled
	}
	// The unique key also serializes the first two deliveries of the same saga.
	changed, err := q.ClaimStockAttempt(ctx, db.ClaimStockAttemptParams{SagaID: attempt.SagaID, OrderID: attempt.OrderID, State: initial})
	if err != nil || changed != 0 {
		return "", err
	}
	row, err := q.LockStockAttempt(ctx, attempt.SagaID)
	if err != nil {
		return "", err
	}
	if row.OrderID != attempt.OrderID {
		return "", errors.New("saga belongs to a different order")
	}
	return row.State, nil
}

func applyCommand(ctx context.Context, tx pgx.Tx, q *db.Queries, next action, command stockmessage.Deduction) error {
	switch next {
	case deduct:
		event, err := deductStock(ctx, tx, q, command)
		if err != nil {
			return err
		}
		return outbox.Add(ctx, tx, command.SagaID, event, command.Attempt)
	case repeatDeducted:
		return outbox.Add(ctx, tx, command.SagaID, stockmessage.StockDeducted, command.Attempt)
	case repeatRejected:
		return outbox.Add(ctx, tx, command.SagaID, stockmessage.StockRejected, command.Attempt)
	case restore:
		return restoreStock(ctx, q, command.SagaID)
	case ignore, tombstone:
		// The initial INSERT already records a cancellation tombstone.
		return nil
	}
	return fmt.Errorf("unknown stock action: %d", next)
}

func deductStock(ctx context.Context, tx pgx.Tx, q *db.Queries, command stockmessage.Deduction) (string, error) {
	deductedAll, err := tryDeduct(ctx, tx, command)
	if err != nil {
		return "", err
	}
	if !deductedAll {
		return stockmessage.StockRejected, nil
	}
	_, err = q.MarkStockAttempt(ctx, db.MarkStockAttemptParams{SagaID: command.SagaID, State: deducted, State_2: rejected})
	return stockmessage.StockDeducted, err
}

func tryDeduct(ctx context.Context, tx pgx.Tx, command stockmessage.Deduction) (bool, error) {
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = savepoint.Rollback(ctx) }()
	q := db.New(savepoint)
	for _, item := range command.Items {
		changed, err := q.DeductItem(ctx, db.DeductItemParams{ID: item.ID, Qty: item.Quantity})
		if err != nil {
			return false, err
		}
		if changed == 0 {
			// Roll back only the item mutations. The rejected attempt and its reply remain.
			return false, savepoint.Rollback(ctx)
		}
		if err := q.RecordDeductedItem(ctx, db.RecordDeductedItemParams{
			SagaID: command.SagaID, ItemID: item.ID, Quantity: item.Quantity,
		}); err != nil {
			return false, err
		}
	}
	return true, savepoint.Commit(ctx)
}

func restoreStock(ctx context.Context, q *db.Queries, id uuid.UUID) error {
	changed, err := q.MarkStockAttempt(ctx, db.MarkStockAttemptParams{SagaID: id, State: cancelled, State_2: deducted})
	if err != nil || changed == 0 {
		return err
	}
	return q.RestoreStock(ctx, id)
}
