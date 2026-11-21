package order

import (
	"context"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/google/uuid"
)

func (h handler) ListOrders(ctx context.Context, req api.ListOrdersRequestObject) (api.ListOrdersResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	params := req.Params
	switch {
	case user.Role == httpx.RoleClient && params.ClientId != nil && *params.ClientId == api.Me && params.Status == nil:
		return h.listClientOrderHistory(ctx, user.ID, params.VehicleId)
	case user.Role != httpx.RoleClient && params.Status != nil && *params.Status == api.ListOrdersParamsStatusRequested &&
		params.ClientId == nil && params.VehicleId == nil:
		return h.listServiceBookings(ctx)
	default:
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "select the booking pool or client order history",
			Fields:  map[string]string{"query": "invalid_filter_combination"},
		}
	}
}

func (h handler) listClientOrderHistory(ctx context.Context, clientID uuid.UUID, vehicleID *uuid.UUID) (api.ListOrdersResponseObject, error) {
	if vehicleID != nil {
		if err := h.requireVehicleOwner(ctx, *vehicleID, clientID); err != nil {
			return nil, err
		}
	}
	rows, err := h.repo.listClientOrderHistory(ctx, clientID, vehicleID)
	if err != nil {
		return nil, err
	}
	response := make(orderListResponse[api.OrderHistoryItem], 0, len(rows))
	for _, row := range rows {
		response = append(response, api.OrderHistoryItem{
			Id:               row.ID,
			VehicleId:        row.VehicleID,
			Status:           api.OrderHistoryItemStatus(row.Status),
			Stage:            api.OrderHistoryItemStage(row.Stage),
			PaymentConfirmed: row.PaymentConfirmed,
			CreatedAt:        row.CreatedAt.Time,
		})
	}
	return response, nil
}

// Оба варианта ответа OpenAPI сериализуются как массив, включая пустой список.
type orderListResponse[T api.OrderHistoryItem | api.ServiceBookingListItem] []T

func (response orderListResponse[T]) VisitListOrdersResponse(w http.ResponseWriter) error {
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}
