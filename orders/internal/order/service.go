package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
)

// AddOrderService получает название и цену из Services до записи позиции в активный заказ.
func (h handler) AddOrderService(ctx context.Context, req api.AddOrderServiceRequestObject) (api.AddOrderServiceResponseObject, error) {
	engineer, err := httpx.EngineerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	service, err := h.services.lookup(ctx, req.Body.ServiceId)
	if errors.Is(err, errCatalogEntryNotFound) {
		return nil, &httpx.Error{Status: http.StatusNotFound, Code: "service_not_found", Message: "service not found"}
	}
	if err != nil {
		h.logger.WarnContext(ctx, "could not get service price", "err", err)
		return nil, &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "services unavailable"}
	}
	id, err := h.repo.addService(ctx, req.Id, engineer.ID, req.Body.ServiceId, req.Body.Quantity, service)
	if err != nil {
		return nil, transitionError(err, "order_not_editable", "order must be active")
	}
	return api.AddOrderService201JSONResponse{
		Id: id, ServiceId: req.Body.ServiceId, Name: service.name,
		Price: service.price.StringFixed(2), Quantity: req.Body.Quantity,
	}, nil
}
