package order

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (h handler) ListClientCertificates(ctx context.Context, _ api.ListClientCertificatesRequestObject) (api.ListClientCertificatesResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	if user.Role != httpx.RoleClient {
		return nil, certificateRoleForbidden()
	}
	rows, err := h.repo.queries.ListClientCertificates(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("list client certificates: %w", err)
	}
	result := make(api.ListClientCertificates200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.ClientCertificate{
			Id: row.ID, OrderId: row.OrderID, CreatedAt: row.CreatedAt.Time,
			Status: certificateStatus(row.Status, row.Attempts, row.NextAttemptAt.Valid),
		})
	}
	return result, nil
}

func readCertificateStatus(ctx context.Context, queries *db.Queries, orderID uuid.UUID) (*api.CertificateStatus, error) {
	row, err := queries.GetOrderCertificateState(ctx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read order certificate status: %w", err)
	}
	status := certificateStatus(row.Status, row.Attempts, row.NextAttemptAt.Valid)
	return &status, nil
}

func certificateStatus(status string, attempts int32, retryScheduled bool) api.CertificateStatus {
	if status == "completed" {
		return "ready"
	}
	// failed с назначенным повтором остаётся в работе, даже если воркер уже может его забрать.
	if status == "failed" && attempts == 3 && !retryScheduled {
		return "failed"
	}
	return "processing"
}

func certificateRoleForbidden() *httpx.Error {
	return &httpx.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "forbidden"}
}
