package certificate

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	queries   *db.Queries
	load      LoadSnapshot
	converter converter
	logger    *slog.Logger
}

// NewWorker подключает загрузку снимка и конвертер. Генерация выполняется фоновым циклом.
func NewWorker(pool *pgxpool.Pool, load LoadSnapshot, converter converter, logger *slog.Logger) *Worker {
	return &Worker{queries: db.New(pool), load: load, converter: converter, logger: logger}
}

// Run проверяет ожидающие справки раз в секунду до отмены контекста.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processDue(ctx)
		}
	}
}

func (w *Worker) processDue(ctx context.Context) {
	rows, err := w.queries.ListDueCertificates(ctx)
	if err != nil {
		if ctx.Err() == nil {
			w.logger.WarnContext(ctx, "list due certificates", "err", err)
		}
		return
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			return
		}
		w.processOne(ctx, row)
	}
}

func (w *Worker) processOne(ctx context.Context, row db.ListDueCertificatesRow) {
	snapshot, pdf, err := w.generate(ctx, row)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		w.logger.WarnContext(ctx, "generate certificate", "order_id", row.OrderID, "err", err)
	}
	if err := saveAttempt(ctx, w.queries, row, snapshot, pdf); err != nil {
		w.logger.WarnContext(ctx, "save certificate attempt", "order_id", row.OrderID, "err", err)
	}
}

// Loading data and running Chromium take place before the conditional UPDATE.
func (w *Worker) generate(ctx context.Context, row db.ListDueCertificatesRow) ([]byte, []byte, error) {
	snapshot, err := w.load(ctx, row.OrderID)
	if err != nil {
		return nil, nil, err
	}
	html, err := render(snapshot)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, nil, err
	}
	pdf, err := w.converter.Convert(ctx, html, map[string][]byte{"DejaVuSans.ttf": font})
	if err != nil {
		return nil, nil, err
	}
	return data, pdf, nil
}
