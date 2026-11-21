package order

import (
	"context"
	"net/http"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/google/uuid"
)

func (h handler) createServiceBooking(ctx context.Context, clientID uuid.UUID, request *api.CreateOrderRequest) (api.CreateOrderResponseObject, error) {
	body, err := request.AsCreateServiceBookingRequest()
	if err != nil {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "invalid service booking", Fields: map[string]string{"body": "invalid"},
		}
	}
	fields := make(map[string]string)
	body.Service = strings.TrimSpace(body.Service)
	if body.Service == "" {
		fields["service"] = "required"
	}
	if body.Date.IsZero() {
		fields["date"] = "required"
	}
	if body.VehicleId == uuid.Nil {
		fields["vehicle_id"] = "required"
	}
	if len(fields) != 0 {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "invalid service booking", Fields: fields,
		}
	}
	if err := h.requireVehicleOwner(ctx, body.VehicleId, clientID); err != nil {
		return nil, err
	}
	id, err := h.repo.createBooking(ctx, clientID, body.VehicleId, body.Service, body.Date.Time, body.Comment)
	if err != nil {
		return nil, err
	}
	return api.CreateOrder201JSONResponse{OrderId: id}, nil
}
