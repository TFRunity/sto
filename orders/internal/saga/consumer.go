package saga

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/stockmessage"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// HandleReply передаётся консьюмеру из cmd/orders/main.go. Разбирает ответ
// Inventory и вызывает applyReply внутри транзакции. RabbitMQ получит Ack
// после возврата nil, включая случай, когда повтор не потребовал записи.
func (s *Service) HandleReply(ctx context.Context, event string, body []byte, logger *slog.Logger) error {
	var attempt stockmessage.Attempt
	if err := json.Unmarshal(body, &attempt); err != nil {
		return err
	}
	if attempt.SagaID == uuid.Nil || attempt.OrderID == uuid.Nil {
		return errors.New("missing saga or order ID")
	}
	return postgres.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		applied, err := applyReply(ctx, db.New(tx), event, attempt)
		if err == nil && !applied {
			logger.WarnContext(ctx, "ignoring stock reply", "saga_id", attempt.SagaID, "event", event)
		}
		return err
	})
}

// applyReply находит попытку по saga_id, проверяет order_id и через replyStep
// выбирает переход. Справку или возврат заказа в active выполняет только после
// успешного transition. Эти записи коммитятся вместе со сменой шага.
func applyReply(ctx context.Context, q *db.Queries, event string, attempt stockmessage.Attempt) (bool, error) {
	row, err := q.GetSaga(ctx, attempt.SagaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	step := replyStep(row.CurrentStep, event)
	if step == "" || row.OrderID != attempt.OrderID {
		return false, nil
	}
	// Ответ с истёкшим сроком пропускаем, даже если воркер ещё не дошёл до попытки.
	changed, err := transition(ctx, q, attempt.SagaID, step)
	if err != nil || !changed {
		return false, err
	}
	if step == completed {
		return true, q.CreateCertificate(ctx, row.OrderID)
	}
	return true, q.RestoreActiveOrder(ctx, row.OrderID)
}
