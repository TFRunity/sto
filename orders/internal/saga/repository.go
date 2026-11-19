package saga

import (
	"context"
	"errors"

	"github.com/AlexeBabanov/sto/chassis/outbox"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/stockmessage"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotActive сообщает PayOrder, что условный UPDATE пропустил заказ.
// HTTP-обработчик отвечает 409, в том числе если заказ принадлежит другому клиенту.
var ErrNotActive = errors.New("order unavailable for payment")

// Service объединяет старт оплаты, обработку ответа и фоновую проверку срока.
type Service struct {
	pool *pgxpool.Pool
}

// New вызывается из cmd/orders/main.go. Там же подключаются HTTP-обработчик,
// обработчик очереди и воркер дедлайнов.
func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// StartPayment вызывается из PayOrder в internal/order/payment.go. Сначала
// переводит active-заказ владельца в paid, затем вызывает enqueueDeduction.
// Обе операции входят в одну транзакцию. Если её не удалось закоммитить,
// HTTP-обработчик не отправит 202.
func (s *Service) StartPayment(ctx context.Context, orderID, clientID uuid.UUID) error {
	return postgres.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		changed, err := q.PayOrder(ctx, db.PayOrderParams{ID: orderID, ClientID: clientID})
		if err != nil {
			return err
		}
		if changed == 0 {
			return ErrNotActive
		}
		attempt := stockmessage.Attempt{SagaID: uuid.New(), OrderID: orderID}
		return enqueueDeduction(ctx, tx, q, attempt)
	})
}

// enqueueDeduction продолжает транзакцию StartPayment. Создаёт попытку
// с дедлайном, группирует количества по товару и записывает DeductStock в outbox.
// Саму отправку в RabbitMQ позже выполнит воркер chassis/outbox.
func enqueueDeduction(ctx context.Context, tx pgx.Tx, q *db.Queries, attempt stockmessage.Attempt) error {
	if err := q.CreateSaga(ctx, db.CreateSagaParams{SagaID: attempt.SagaID, OrderID: attempt.OrderID}); err != nil {
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
