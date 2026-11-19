package order

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler struct {
	repo      repository
	inventory inventoryClient
	logger    *slog.Logger
}

type engineerIDKey struct{}

func NewHTTPHandler(pool *pgxpool.Pool, inventoryURL url.URL, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, err
	}
	h := handler{
		repo:      repository{pool: pool, queries: db.New(pool)},
		inventory: inventoryClient{baseURL: inventoryURL, http: &http.Client{Timeout: 5 * time.Second}},
		logger:    logger,
	}
	handleError := httpx.StrictErrorHandler(logger)
	strict := api.NewStrictHandlerWithOptions(h, []api.StrictMiddlewareFunc{readEngineerID(logger)}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  handleError,
		ResponseErrorHandlerFunc: handleError,
	})
	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}

func readEngineerID(logger *slog.Logger) api.StrictMiddlewareFunc {
	return func(next api.StrictHandlerFunc, operation string) api.StrictHandlerFunc {
		if operation != "CreateOrder" {
			return next
		}
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			id, err := uuid.Parse(r.Header.Get("X-User-Id"))
			if err != nil || id == uuid.Nil {
				logger.WarnContext(ctx, "missing or invalid engineer ID")
				return nil, &httpx.Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "unauthorized"}
			}
			return next(context.WithValue(ctx, engineerIDKey{}, id), w, r, request)
		}
	}
}

func (h handler) CreateOrder(ctx context.Context, req api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	engineerID, ok := ctx.Value(engineerIDKey{}).(uuid.UUID)
	if !ok {
		return nil, &httpx.Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "unauthorized"}
	}
	id, err := h.repo.create(ctx, req.Body.ClientId, req.Body.VehicleId, engineerID)
	if err != nil {
		return nil, err
	}
	return api.CreateOrder201JSONResponse{OrderId: id}, nil
}

func (h handler) AddOrderItem(ctx context.Context, req api.AddOrderItemRequestObject) (api.AddOrderItemResponseObject, error) {
	price, err := h.inventory.price(ctx, req.Body.ItemId)
	if errors.Is(err, errItemNotFound) {
		return nil, &httpx.Error{Status: http.StatusNotFound, Code: "item_not_found", Message: "item not found"}
	}
	if err != nil {
		h.logger.WarnContext(ctx, "could not get inventory price", "err", err)
		return nil, &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "inventory unavailable"}
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

func orderNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "order_not_found", Message: "order not found"}
}

type itemJSON struct {
	ID       uuid.UUID `json:"id"`
	ItemID   uuid.UUID `json:"item_id"`
	Quantity int32     `json:"quantity"`
	Price    string    `json:"price"`
}

func itemResponse(item orderItem) itemJSON {
	return itemJSON{ID: item.id, ItemID: item.itemID, Quantity: item.quantity, Price: item.price.StringFixed(2)}
}

func (item itemJSON) VisitAddOrderItemResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(item)
}

func (h handler) GetOrder(ctx context.Context, req api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	row, items, err := h.repo.get(ctx, req.Id)
	if errors.Is(err, errOrderNotFound) {
		return nil, orderNotFound()
	}
	if err != nil {
		return nil, err
	}

	response := orderJSON{
		Order: api.Order{
			Id: row.ID, ClientId: row.ClientID, VehicleId: row.VehicleID,
			EngineerId: row.EngineerID, Status: api.OrderStatus(row.Status),
			Stage: api.OrderStage(row.Stage), PaymentConfirmed: row.PaymentConfirmed,
		},
		Items: make([]itemJSON, 0, len(items)),
		Total: total(items).StringFixed(2),
	}
	for _, item := range items {
		response.Items = append(response.Items, itemResponse(item))
	}
	return response, nil
}

type orderJSON struct {
	api.Order
	Items []itemJSON `json:"items"`
	Total string     `json:"total"`
}

func (order orderJSON) VisitGetOrderResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(order)
}
