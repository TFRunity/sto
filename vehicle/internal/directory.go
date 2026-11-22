package vehicle

import (
	"context"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/vehicle/internal/api"
)

// ListClientVehicles возвращает автомобили владельца, включая ТС без заказов.
func (h *Handler) ListClientVehicles(ctx context.Context, _ api.ListClientVehiclesRequestObject) (api.ListClientVehiclesResponseObject, error) {
	identity, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}

	rows, err := h.repo.queries.ListClientVehicles(ctx, identity.ID)
	if err != nil {
		return nil, err
	}

	result := make(api.ListClientVehicles200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.ClientVehicleListItem{
			VehicleId: row.VehicleID,
			Number:    row.Number,
			Brand:     row.Brand,
		})
	}
	return result, nil
}

// ListVehicleMakes читает справочник для выбора марки при добавлении ТС.
func (h *Handler) ListVehicleMakes(ctx context.Context, _ api.ListVehicleMakesRequestObject) (api.ListVehicleMakesResponseObject, error) {
	if _, ok := httpx.IdentityFromContext(ctx); !ok {
		return nil, httpx.Unauthorized()
	}

	rows, err := h.repo.queries.ListVehicleMakes(ctx)
	if err != nil {
		return nil, err
	}

	result := make(api.ListVehicleMakes200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.VehicleMake{Name: row.Name, Aliases: row.Aliases})
	}
	return result, nil
}

// SearchVehicles нормализует номер так же, как при записи ТС. Поиск включает историю.
func (h *Handler) SearchVehicles(ctx context.Context, req api.SearchVehiclesRequestObject) (api.SearchVehiclesResponseObject, error) {
	number := normalizePlate(req.Params.Q)
	if number == "" {
		return api.SearchVehicles200JSONResponse{}, nil
	}
	ids, err := h.repo.queries.SearchVehicles(ctx, postgres.ContainsPattern(number))
	if err != nil {
		return nil, err
	}
	if len(ids) > httpx.SearchLimit {
		return nil, httpx.SearchTooBroad()
	}
	return append(api.SearchVehicles200JSONResponse{}, ids...), nil
}

// GetVehiclesByIDs возвращает номера по существующим ID, в том числе у мягко удалённых ТС.
func (h *Handler) GetVehiclesByIDs(ctx context.Context, req api.GetVehiclesByIDsRequestObject) (api.GetVehiclesByIDsResponseObject, error) {
	rows, err := h.repo.queries.GetVehiclesByIDs(ctx, req.Params.Ids)
	if err != nil {
		return nil, err
	}
	result := make(api.GetVehiclesByIDs200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.VehicleSummary{Id: row.ID, Number: row.Number})
	}
	return result, nil
}
