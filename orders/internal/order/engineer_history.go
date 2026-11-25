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

func (h handler) ListEngineerOrders(ctx context.Context, req api.ListEngineerOrdersRequestObject) (api.ListEngineerOrdersResponseObject, error) {
	engineer, err := httpx.EngineerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.repo.queries.ListAssignedOrders(ctx, db.ListAssignedOrdersParams{
		EngineerID: &engineer.ID,
		Completed:  req.Params.Scope == api.ListEngineerOrdersParamsScopeCompleted,
	})
	if err != nil {
		return nil, fmt.Errorf("list assigned orders: %w", err)
	}
	result := make(engineerOrderHistory, 0, len(rows))
	for _, row := range rows {
		result = append(result, orderHistoryItem(db.ListClientOrderHistoryRow(row)))
	}
	return result, nil
}

func (h handler) ListVehicleHistoryForEngineer(ctx context.Context, req api.ListVehicleHistoryForEngineerRequestObject) (api.ListVehicleHistoryForEngineerResponseObject, error) {
	engineer, err := httpx.EngineerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.repo.listVehicleHistoryForEngineer(ctx, engineer.ID, req.Id)
	if errors.Is(err, errOrderNotFound) {
		return nil, orderNotFound()
	}
	if err != nil {
		return nil, err
	}
	result := make(engineerOrderHistory, 0, len(rows))
	for _, row := range rows {
		result = append(result, orderHistoryItem(db.ListClientOrderHistoryRow(row)))
	}
	return result, nil
}

type engineerOrderHistory []api.OrderHistoryItem

func (history engineerOrderHistory) VisitListEngineerOrdersResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusOK, history)
	return nil
}

func (history engineerOrderHistory) VisitListVehicleHistoryForEngineerResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusOK, history)
	return nil
}

func (r repository) listVehicleHistoryForEngineer(ctx context.Context, engineerID, orderID uuid.UUID) ([]db.ListVehicleOrderHistoryRow, error) {
	var rows []db.ListVehicleOrderHistoryRow
	// Назначение и история читаются из одного снимка: другой инженер может забрать заявку.
	err := postgres.InReadSnapshot(ctx, r.pool, func(tx pgx.Tx) error {
		queries := r.queries.WithTx(tx)
		vehicleID, err := vehicleForEngineerHistory(ctx, queries, engineerID, orderID)
		if err != nil {
			return err
		}
		rows, err = queries.ListVehicleOrderHistory(ctx, db.ListVehicleOrderHistoryParams{
			VehicleID: vehicleID, CurrentOrderID: orderID,
		})
		if err != nil {
			return fmt.Errorf("read vehicle order history: %w", err)
		}
		return nil
	})
	return rows, err
}

func vehicleForEngineerHistory(ctx context.Context, queries *db.Queries, engineerID, orderID uuid.UUID) (uuid.UUID, error) {
	vehicleID, err := queries.GetVehicleForEngineerHistory(ctx, db.GetVehicleForEngineerHistoryParams{
		OrderID: orderID, EngineerID: &engineerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errOrderNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("check engineer vehicle history access: %w", err)
	}
	return vehicleID, nil
}

func (h handler) GetHistoricalOrderForEngineer(ctx context.Context, req api.GetHistoricalOrderForEngineerRequestObject) (api.GetHistoricalOrderForEngineerResponseObject, error) {
	engineer, err := httpx.EngineerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	return h.readOrder(ctx, req.HistoricalId, orderRead{
		engineerID: engineer.ID, historyAnchorID: req.Id, includeEngineers: true,
	})
}
