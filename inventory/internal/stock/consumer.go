package stock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AlexeBabanov/sto/chassis/stockmessage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewHandler returns the command handler for the inventory queue.
func NewHandler(pool *pgxpool.Pool) func(context.Context, string, []byte) error {
	return repository{pool: pool}.handle
}

func (r repository) handle(ctx context.Context, event string, body []byte) error {
	if event != stockmessage.DeductStock && event != stockmessage.CancelStock {
		return fmt.Errorf("unknown stock command: %s", event)
	}
	var command stockmessage.Deduction
	if err := json.Unmarshal(body, &command); err != nil {
		return err
	}
	if command.SagaID == uuid.Nil || command.OrderID == uuid.Nil {
		return errors.New("missing saga or order ID")
	}
	for _, item := range command.Items {
		if item.ID == uuid.Nil || item.Quantity <= 0 {
			return errors.New("invalid deduction item")
		}
	}
	return r.apply(ctx, event, command)
}
