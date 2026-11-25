package internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/gateway/internal/api"
	"github.com/google/uuid"
)

type engineerOrders struct {
	client  *http.Client
	orders  url.URL
	vehicle url.URL
	logger  *slog.Logger
}

// ListEngineerOrders сначала получает доступные инженеру заказы, затем запрашивает подписи только для ТС из этого списка.
func (h engineerOrders) ListEngineerOrders(ctx context.Context, req api.ListEngineerOrdersRequestObject) (api.ListEngineerOrdersResponseObject, error) {
	engineer, ok := IdentityFromContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	orders, err := h.loadOrders(ctx, engineer, req.Params.Scope)
	if err != nil {
		return nil, err
	}
	h.addVehicleNames(ctx, orders)
	return engineerOrderList(orders), nil
}

func (h engineerOrders) loadOrders(ctx context.Context, engineer Identity, scope api.ListEngineerOrdersParamsScope) ([]api.OrderHistoryItem, error) {
	params := url.Values{"scope": {string(scope)}}
	orders, _, err := httpx.GetJSONWithHeaders[[]api.OrderHistoryItem](ctx, h.client, h.orders,
		"/engineer/orders", params, upstreamIdentityHeaders(engineer))
	if err == nil {
		return orders, nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, &httpx.Error{Status: http.StatusGatewayTimeout, Code: "upstream_timeout", Message: "orders unavailable"}
	}
	h.logger.WarnContext(ctx, "could not load engineer orders", "err", err)
	return nil, &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "orders unavailable"}
}

func (h engineerOrders) addVehicleNames(ctx context.Context, orders []api.OrderHistoryItem) {
	ids := uniqueVehicleIDs(orders)
	vehicles := make(map[uuid.UUID]vehicleSummary, len(ids))
	// Vehicle ограничивает пачку 200 ID. Дедлайн общий для чтения заказов и всех пачек.
	for batch := range slices.Chunk(ids, httpx.SearchLimit) {
		params := url.Values{"ids": batch}
		found, _, err := httpx.GetJSON[[]vehicleSummary](ctx, h.client, h.vehicle, "/internal/vehicles", params)
		if err != nil {
			// Сбой справочника не скрывает сами заказы.
			h.logger.WarnContext(ctx, "could not load vehicles for engineer orders", "err", err)
			break
		}
		for _, vehicle := range found {
			vehicles[vehicle.ID] = vehicle
		}
	}
	for i := range orders {
		if vehicle, found := vehicles[orders[i].VehicleId]; found {
			orders[i].VehicleNumber = &vehicle.Number
			orders[i].VehicleBrand = vehicle.Brand
		}
	}
}

func uniqueVehicleIDs(orders []api.OrderHistoryItem) []string {
	seen := make(map[uuid.UUID]bool, len(orders))
	ids := make([]string, 0, len(orders))
	for _, order := range orders {
		if !seen[order.VehicleId] {
			ids = append(ids, order.VehicleId.String())
			seen[order.VehicleId] = true
		}
	}
	return ids
}

type engineerOrderList []api.OrderHistoryItem

func (orders engineerOrderList) VisitListEngineerOrdersResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusOK, orders)
	return nil
}
