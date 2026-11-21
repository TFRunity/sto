package vehicle

import (
	"context"
	"errors"

	"github.com/AlexeBabanov/sto/vehicle/internal/api"
)

// CheckVehicleOwnership обслуживает внутреннюю проверку принадлежности ТС для Orders.
func (h *Handler) CheckVehicleOwnership(ctx context.Context, req api.CheckVehicleOwnershipRequestObject) (api.CheckVehicleOwnershipResponseObject, error) {
	row, err := h.repo.get(ctx, req.Id)
	if errors.Is(err, errVehicleNotFound) {
		return nil, vehicleNotFound()
	}
	if err != nil {
		return nil, err
	}
	if row.OwnerID != req.Params.OwnerId {
		return nil, vehicleNotFound()
	}
	return api.CheckVehicleOwnership204Response{}, nil
}
