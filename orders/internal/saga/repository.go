package saga

import (
	"context"
	"errors"
	"time"

	"github.com/AlexeBabanov/sto/chassis/outbox"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/stockmessage"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotActive возвращается для заказа без доступной клиенту отметки оплаты.
var ErrNotActive = errors.New("order unavailable for payment")

var ErrStockRetryUnavailable = errors.New("stock deduction cannot be retried")

// Service подтверждает оплату и ведёт попытки списания.
type Service struct {
	pool *pgxpool.Pool
}

// New вызывается из cmd/orders/main.go. Там же подключаются HTTP-обработчик,
// обработчик очереди и воркер дедлайнов.
func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ConfirmPayment фиксирует оплату и команду списания в одной транзакции.
// Если переход уже выполнился, возвращает сохранённое время оплаты владельцу.
func (s *Service) ConfirmPayment(ctx context.Context, orderID, clientID uuid.UUID) (time.Time, error) {
	var confirmedAt time.Time
	err := postgres.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		paidAt, err := q.ConfirmOrderPayment(ctx, db.ConfirmOrderPaymentParams{ID: orderID, ClientID: clientID})
		if errors.Is(err, pgx.ErrNoRows) {
			// UPDATE дожидается конкурирующего перехода. Следующий SELECT
			// видит его зафиксированный результат и не создаёт новую попытку.
			confirmedAt, err = readConfirmedPaymentTime(ctx, q, orderID, clientID)
			return err
		}
		if err != nil {
			return err
		}
		confirmedAt = paidAt.Time
		attempt := stockmessage.Attempt{SagaID: uuid.New(), OrderID: orderID}
		return enqueueDeduction(ctx, tx, q, attempt)
	})
	return confirmedAt, err
}

func readConfirmedPaymentTime(ctx context.Context, q *db.Queries, orderID, clientID uuid.UUID) (time.Time, error) {
	paidAt, err := q.GetConfirmedPaymentTime(ctx, db.GetConfirmedPaymentTimeParams{ID: orderID, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrNotActive
	}
	return paidAt.Time, err
}

// RetryStock сериализует повторы одного заказа блокировкой строки Orders.
func (s *Service) RetryStock(ctx context.Context, orderID, engineerID uuid.UUID) error {
	return postgres.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := checkStockRetry(ctx, q, orderID, engineerID); err != nil {
			return err
		}
		return enqueueDeduction(ctx, tx, q, stockmessage.Attempt{SagaID: uuid.New(), OrderID: orderID})
	})
}

func checkStockRetry(ctx context.Context, q *db.Queries, orderID, engineerID uuid.UUID) error {
	_, err := q.LockPaidOrderForStockRetry(ctx, db.LockPaidOrderForStockRetryParams{
		OrderID: orderID, EngineerID: engineerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrStockRetryUnavailable
	}
	if err != nil {
		return err
	}
	// Проверяем отдельным запросом после блокировки, чтобы увидеть попытку,
	// которую мог записать предыдущий повтор, пока этот ждал строку заказа.
	allowed, err := q.CanRetryStockDeduction(ctx, orderID)
	if err != nil {
		return err
	}
	if allowed == nil || !*allowed {
		return ErrStockRetryUnavailable
	}
	return nil
}

// enqueueDeduction создаёт попытку, группирует количества по товару
// и записывает DeductStock в outbox.
// Саму отправку в RabbitMQ позже выполнит воркер chassis/outbox.
func enqueueDeduction(ctx context.Context, tx pgx.Tx, q *db.Queries, attempt stockmessage.Attempt) error {
	if err := q.CreateStockAttempt(ctx, db.CreateStockAttemptParams{SagaID: attempt.SagaID, OrderID: attempt.OrderID}); err != nil {
		return err
	}
	rows, err := q.ListDeductionItems(ctx, attempt.OrderID)
	if err != nil {
		return err
	}
	items := make([]stockmessage.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, stockmessage.Item{ID: row.ItemID, Quantity: row.Quantity})
	}
	return outbox.Add(ctx, tx, attempt.SagaID, stockmessage.DeductStock, stockmessage.Deduction{Attempt: attempt, Items: items})
}

// transition вызывается из applyReply. Условный UPDATE из queries/saga.sql
// проверяет шаг и срок в момент записи. false при nil-ошибке означает, что
// ответ уже нельзя применить. В этом случае applyReply не меняет заказ и справку.
func transition(ctx context.Context, q *db.Queries, id uuid.UUID, step string) (bool, error) {
	changed, err := q.TransitionSaga(ctx, db.TransitionSagaParams{SagaID: id, CurrentStep: step})
	return changed != 0, err
}
