package certificate

import (
	"context"
	"time"

	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func saveAttempt(ctx context.Context, queries *db.Queries, row db.ListDueCertificatesRow, snapshot, pdf []byte) error {
	attempts := row.Attempts + 1
	status, next := nextAttempt(attempts, len(pdf) != 0, time.Now())
	_, err := queries.SaveCertificateAttempt(ctx, db.SaveCertificateAttemptParams{
		ID: row.ID, OldStatus: row.Status, OldAttempts: row.Attempts,
		NewStatus: status, NewAttempts: attempts, NextAttemptAt: pgtype.Timestamptz{Time: next, Valid: !next.IsZero()},
		Snapshot: snapshot, Pdf: pdf,
	})
	return err
}
