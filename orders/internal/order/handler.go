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

type identityKey struct{}
type identity struct {
	id   uuid.UUID
	role string
}

func requestIdentity(ctx context.Context) (identity, bool) {
	user, ok := ctx.Value(identityKey{}).(identity)
	return user, ok
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
	strict := api.NewStrictHandlerWithOptions(h, []api.StrictMiddlewareFunc{readIdentity(h.logger)}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handleError,
		ResponseErrorHandlerFunc: handleError,
	})
	return api.Handler(strict)
}

// Strict-обработчик получает context и тело. Заголовок читаем на HTTP-границе
// и передаём дальше только проверенный UUID.
func readIdentity(logger *slog.Logger) api.StrictMiddlewareFunc {
	return func(next api.StrictHandlerFunc, _ string) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			id, err := uuid.Parse(r.Header.Get(httpx.UserIDHeader))
			role := r.Header.Get(httpx.UserRoleHeader)
			if err != nil || id == uuid.Nil || role == "" {
				logger.WarnContext(ctx, "missing or invalid request identity")
				return nil, unauthorized()
			}
			return next(context.WithValue(ctx, identityKey{}, identity{id: id, role: role}), w, r, request)
		}
	}
}

func (h handler) CreateOrder(ctx context.Context, req api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	user, ok := requestIdentity(ctx)
	if !ok {
		return nil, unauthorized()
	}
	id, err := h.repo.create(ctx, req.Body.ClientId, req.Body.VehicleId, user.id)
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

func unauthorized() *httpx.Error {
	return &httpx.Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "unauthorized"}
}

func orderNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "order_not_found", Message: "order not found"}
}

func (h handler) GetOrder(ctx context.Context, req api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	row, items, err := h.repo.get(ctx, req.Id)
	if errors.Is(err, errOrderNotFound) {
		return nil, orderNotFound()
	}
	if err != nil {
		return nil, err
	}

	user, ok := requestIdentity(ctx)
	if !ok {
		return nil, unauthorized()
	}
	if user.role == "client" && row.ClientID != user.id {
		return nil, orderNotFound()
	}
	return orderResponse(row, items), nil
}
