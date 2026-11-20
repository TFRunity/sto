package order

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/AlexeBabanov/sto/orders/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type handler struct {
	repo      repository
	payments  *saga.Service
	inventory inventoryClient
	logger    *slog.Logger
}

// NewHTTPHandler подключает валидацию OpenAPI перед сгенерированным HTTP-сервером.
func NewHTTPHandler(pool *pgxpool.Pool, inventoryURL url.URL, payments *saga.Service, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, err
	}
	h := newHandler(pool, inventoryURL, payments, logger)
	return httpx.Validator(spec, logger)(strictHTTPHandler(h)), nil
}

func newHandler(pool *pgxpool.Pool, inventoryURL url.URL, payments *saga.Service, logger *slog.Logger) handler {
	return handler{
		payments:  payments,
		repo:      repository{pool: pool, queries: db.New(pool)},
		inventory: newInventoryClient(inventoryURL),
		logger:    logger,
	}
}

func strictHTTPHandler(h handler) http.Handler {
	handleError := httpx.StrictErrorHandler(h.logger)
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  httpx.StrictRequestErrorHandler(h.logger),
		ResponseErrorHandlerFunc: handleError,
	})
	return httpx.RequireIdentity(api.Handler(strict), h.logger)
}

func (h handler) CreateOrder(ctx context.Context, req api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	id, err := h.repo.create(ctx, req.Body.ClientId, req.Body.VehicleId, user.ID)
	if err != nil {
		return nil, err
	}
	return api.CreateOrder201JSONResponse{OrderId: id}, nil
}

func (h handler) AddOrderItem(ctx context.Context, req api.AddOrderItemRequestObject) (api.AddOrderItemResponseObject, error) {
	price, err := h.itemPrice(ctx, req.Body.ItemId)
	if err != nil {
		return nil, err
	}

	item := orderItem{itemID: req.Body.ItemId, quantity: req.Body.Quantity, price: price}
	item.id, err = h.repo.addItem(ctx, req.Id, item)
	if errors.Is(err, errOrderNotFound) {
		return nil, orderNotFound()
	}
	if err != nil {
		return nil, err
	}
	return itemResponse(item), nil
}

func (h handler) itemPrice(ctx context.Context, itemID uuid.UUID) (decimal.Decimal, error) {
	product, err := h.inventory.lookup(ctx, itemID)
	if errors.Is(err, errItemNotFound) {
		return decimal.Zero, &httpx.Error{Status: http.StatusNotFound, Code: "item_not_found", Message: "item not found"}
	}
	if err != nil {
		h.logger.WarnContext(ctx, "could not get inventory price", "err", err)
		return decimal.Zero, &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "inventory unavailable"}
	}

	return product.price, nil
}

func orderNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "order_not_found", Message: "order not found"}
}

func (h handler) GetOrder(ctx context.Context, req api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	row, items, err := h.repo.get(ctx, req.Id)
	if errors.Is(err, errOrderNotFound) {
		return nil, orderNotFound()
	}
	if err != nil {
		return nil, err
	}

	if user.Role == httpx.RoleClient && row.ClientID != user.ID {
		return nil, orderNotFound()
	}
	return orderResponse(row, items), nil
}
