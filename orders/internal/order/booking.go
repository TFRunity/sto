package order

import (
	"context"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/google/uuid"
)

func (h handler) createServiceBooking(ctx context.Context, clientID uuid.UUID, request *api.CreateOrderRequest) (api.CreateOrderResponseObject, error) {
	body, err := decodeBooking(request)
	if err != nil {
		return nil, err
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

// decodeBooking adds role-specific checks after the shared OpenAPI union is decoded.
func decodeBooking(request *api.CreateOrderRequest) (api.CreateServiceBookingRequest, error) {
	body, err := request.AsCreateServiceBookingRequest()
	if err != nil {
		return api.CreateServiceBookingRequest{}, &httpx.Error{
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
	} else {
		location, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			return api.CreateServiceBookingRequest{}, err
		}
		year, month, day := time.Now().In(location).Date()
		today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if body.Date.Before(today) {
			fields["date"] = "past_date"
		}
	}
	if body.VehicleId == uuid.Nil {
		fields["vehicle_id"] = "required"
	}
	if len(fields) != 0 {
		return api.CreateServiceBookingRequest{}, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "invalid service booking", Fields: fields,
		}
	}
	return body, nil
}
