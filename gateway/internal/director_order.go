package internal

import (
	"context"
	"net/http"
	"net/url"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/gateway/internal/api"
	"golang.org/x/sync/errgroup"
)

// GetDirectorOrder читает снимок Orders и отдельно получает подписи связанных записей.
func (h director) GetDirectorOrder(ctx context.Context, req api.GetDirectorOrderRequestObject) (api.GetDirectorOrderResponseObject, error) {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return nil, ErrUnauthorized
	}
	if identity.Role != roleDirector {
		return nil, &httpx.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "forbidden"}
	}
	ctx, cancel := context.WithTimeout(ctx, directorTimeout)
	defer cancel()

	order, status, err := httpx.GetJSONWithHeaders[api.Order](
		ctx, h.client, h.orders, "/orders/"+req.Id.String(), nil, upstreamIdentityHeaders(identity),
	)
	if status == http.StatusNotFound {
		return nil, &httpx.Error{Status: http.StatusNotFound, Code: "order_not_found", Message: "order not found"}
	}
	if err != nil {
		return nil, h.sourceError(ctx, err)
	}
	details, err := h.loadOrderReferences(ctx, order)
	if err != nil {
		return nil, h.sourceError(ctx, err)
	}
	return api.GetDirectorOrder200JSONResponse(details), nil
}

func (h director) loadOrderReferences(ctx context.Context, order api.Order) (api.DirectorOrderDetails, error) {
	var people []api.PersonSummary
	var vehicles []api.VehicleSummary
	group, work := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		people, _, err = httpx.GetJSON[[]api.PersonSummary](work, h.client, h.users, "/users",
			url.Values{"ids": {order.ClientId.String()}})
		return err
	})
	group.Go(func() error {
		var err error
		vehicles, _, err = httpx.GetJSON[[]api.VehicleSummary](work, h.client, h.vehicle, "/internal/vehicles",
			url.Values{"ids": {order.VehicleId.String()}})
		return err
	})
	if err := group.Wait(); err != nil {
		return api.DirectorOrderDetails{}, err
	}
	result := api.DirectorOrderDetails{Order: order}
	// Старый заказ остаётся читаемым, даже если одна из связанных записей отсутствует.
	for _, person := range people {
		if person.Id == order.ClientId {
			result.Client = &person
			break
		}
	}
	for _, vehicle := range vehicles {
		if vehicle.Id == order.VehicleId {
			result.Vehicle = &vehicle
			break
		}
	}
	return result, nil
}

func upstreamIdentityHeaders(identity Identity) http.Header {
	headers := http.Header{}
	headers.Set(httpx.UserIDHeader, identity.UserID)
	headers.Set(httpx.UserRoleHeader, identity.Role)
	return headers
}
