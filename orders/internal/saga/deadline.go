package saga

import (
	"context"
	"log/slog"
	"time"

	"github.com/AlexeBabanov/sto/chassis/outbox"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/chassis/stockmessage"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/jackc/pgx/v5"
)

// RunDeadlines запускается из cmd/orders/main.go параллельно с HTTP-сервером
// и консьюмером. Раз в секунду передаёт работу в expireDue. Этот интервал
// задаёт частоту поиска. Сам срок оплаты хранится в строке saga_state.
func (s *Service) RunDeadlines(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.expireDue(ctx, logger)
		}
	}
}

// expireDue читает пачку просроченных попыток и вызывает expire для каждой.
// Ошибка одной попытки записывается в лог, остальные продолжают обрабатываться.
func (s *Service) expireDue(ctx context.Context, logger *slog.Logger) {
	rows, err := db.New(s.pool).ListExpiredSagas(ctx)
	if err != nil {
		if ctx.Err() == nil {
			logger.WarnContext(ctx, "list expired payments", "err", err)
		}
		return
	}
	for _, row := range rows {
		if err := s.expire(ctx, stockmessage.Attempt{SagaID: row.SagaID, OrderID: row.OrderID}); err != nil && ctx.Err() == nil {
			logger.WarnContext(ctx, "expire payment", "saga_id", row.SagaID, "err", err)
		}
	}
}

// expire заново проверяет состояние и срок условным UPDATE. Если переход
// состоялся, в той же транзакции возвращает заказ в active и записывает
// CancelStock в outbox. При нуле обновлённых строк эти действия пропускаются.
func (s *Service) expire(ctx context.Context, attempt stockmessage.Attempt) error {
	return postgres.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		changed, err := q.ExpireSaga(ctx, attempt.SagaID)
		if err != nil || changed == 0 {
			return err
		}
		if err := q.RestoreActiveOrder(ctx, attempt.OrderID); err != nil {
			return err
		}
		return outbox.Add(ctx, tx, attempt.SagaID, stockmessage.CancelStock, attempt)
	})
}
