package vehicle

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/vehicle/internal/api"
	"github.com/jackc/pgx/v5"
)

func (h *Handler) FindVehicleForIntake(ctx context.Context, req api.FindVehicleForIntakeRequestObject) (api.FindVehicleForIntakeResponseObject, error) {
	if _, err := httpx.EngineerIdentity(ctx); err != nil {
		return nil, err
	}
	number := normalizePlate(req.Params.Number)
	if number == "" {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "number is required",
			Fields: map[string]string{"number": "required"},
		}
	}
	row, err := h.repo.queries.FindVehicleForIntake(ctx, number)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, vehicleNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("look up vehicle for intake: %w", err)
	}
	return api.FindVehicleForIntake200JSONResponse{
		VehicleId: row.ID, OwnerId: row.OwnerID, Number: row.Number, Brand: row.Brand,
	}, nil
}

func (h *Handler) CreateVehicleForIntake(ctx context.Context, req api.CreateVehicleForIntakeRequestObject) (api.CreateVehicleForIntakeResponseObject, error) {
	if _, err := httpx.EngineerIdentity(ctx); err != nil {
		return nil, err
	}
	body := req.Body
	vehicle, err := h.createVehicleForOwner(ctx, body.OwnerId, body.Number, body.Brand, body.Vin)
	if err != nil {
		return nil, err
	}
	return api.CreateVehicleForIntake201JSONResponse(vehicle), nil
}
