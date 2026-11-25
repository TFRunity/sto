package order

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var errIntakeRequestConflict = errors.New("intake request uses another client or vehicle")

func (h handler) createEngineerOrder(ctx context.Context, input api.CreateEngineerOrderRequest, engineerID uuid.UUID) (uuid.UUID, error) {
	if input.RequestId != nil && *input.RequestId == uuid.Nil {
		return uuid.Nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "request_id must not be nil",
			Fields: map[string]string{"request_id": "invalid"},
		}
	}
	params := db.CreateOrderParams{
		ClientID: input.ClientId, VehicleID: input.VehicleId, EngineerID: engineerID, RequestID: input.RequestId,
	}
	// Успешный повтор возвращаем до обращения к Vehicle. Машину могли удалить или сервис мог стать недоступен.
	id, err := findIntakeOrder(ctx, h.repo.queries, params)
	if err != nil || id != uuid.Nil {
		return id, err
	}
	if err := h.requireVehicleOwner(ctx, input.VehicleId, input.ClientId); err != nil {
		return uuid.Nil, err
	}
	return h.repo.createIntakeOrder(ctx, params)
}

func (r repository) createIntakeOrder(ctx context.Context, params db.CreateOrderParams) (uuid.UUID, error) {
	var id uuid.UUID
	err := postgres.InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var err error
		id, err = insertIntakeOrder(ctx, r.queries.WithTx(tx), params)
		return err
	})
	return id, err
}

func insertIntakeOrder(ctx context.Context, queries *db.Queries, params db.CreateOrderParams) (uuid.UUID, error) {
	id, err := queries.CreateOrder(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		// При параллельном повторе INSERT ждёт владельца ключа. Отдельный SELECT в READ COMMITTED видит его результат.
		id, err := findIntakeOrder(ctx, queries, params)
		if err == nil && id == uuid.Nil {
			return uuid.Nil, errors.New("intake order missing after request conflict")
		}
		return id, err
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert order: %w", err)
	}
	// Назначение исполнителя и новый заказ фиксируются одной транзакцией.
	if err := queries.AddOrderEngineer(ctx, db.AddOrderEngineerParams{OrderID: id, EngineerID: params.EngineerID}); err != nil {
		return uuid.Nil, fmt.Errorf("assign order engineer: %w", err)
	}
	return id, nil
}

func findIntakeOrder(ctx context.Context, queries *db.Queries, params db.CreateOrderParams) (uuid.UUID, error) {
	if params.RequestID == nil {
		return uuid.Nil, nil
	}
	order, err := queries.GetIntakeOrder(ctx, db.GetIntakeOrderParams{
		EngineerID: params.EngineerID, RequestID: *params.RequestID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("read intake order: %w", err)
	}
	if order.ClientID != params.ClientID || order.VehicleID != params.VehicleID {
		return uuid.Nil, errIntakeRequestConflict
	}
	return order.ID, nil
}

func intakeError(err error) error {
	if errors.Is(err, errIntakeRequestConflict) {
		return &httpx.Error{
			Status: http.StatusConflict, Code: "intake_request_conflict", Message: "request_id already used with different data",
		}
	}
	return err
}
