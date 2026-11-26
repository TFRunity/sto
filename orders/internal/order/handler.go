package order

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/db"
	"github.com/AlexeBabanov/sto/orders/internal/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler struct {
	repo      repository
	payments  *saga.Service
	inventory catalogClient
	vehicles  vehicleClient
	services  catalogClient
	people    peopleDirectory
	logger    *slog.Logger
}

const directorRole = "director"

// NewHTTPHandler подключает валидацию OpenAPI перед сгенерированным HTTP-сервером.
func NewHTTPHandler(pool *pgxpool.Pool, inventoryURL, vehicleURL, servicesURL, usersURL url.URL, payments *saga.Service, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, err
	}
	h := newHandler(pool, inventoryURL, vehicleURL, servicesURL, usersURL, payments, logger)
	return httpx.Validator(spec, logger)(strictHTTPHandler(h)), nil
}

func newHandler(pool *pgxpool.Pool, inventoryURL, vehicleURL, servicesURL, usersURL url.URL, payments *saga.Service, logger *slog.Logger) handler {
	return handler{
		payments:  payments,
		repo:      repository{pool: pool, queries: db.New(pool)},
		inventory: newCatalogClient(inventoryURL, "inventory"),
		vehicles:  newVehicleClient(vehicleURL),
		services:  newCatalogClient(servicesURL, "internal/services"),
		people:    peopleDirectory{baseURL: usersURL, http: &http.Client{Timeout: 3 * time.Second}},
		logger:    logger,
	}
}

func strictHTTPHandler(h handler) http.Handler {
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions(httpx.NewStrictErrors(h.logger)))
	routes := api.Handler(strict)
	mux := http.NewServeMux()
	mux.Handle("GET /orders/stats", routes)
	mux.Handle("GET /internal/orders", routes)
	mux.Handle("GET /internal/orders/{id}/chat-participants", routes)
	mux.Handle("/", httpx.RequireIdentity(routes, h.logger))
	return mux
}

// GetOrderChatParticipants читает владельца и ответственного одним запросом.
func (h handler) GetOrderChatParticipants(ctx context.Context, req api.GetOrderChatParticipantsRequestObject) (api.GetOrderChatParticipantsResponseObject, error) {
	participants, err := h.repo.chatParticipants(ctx, req.Id)
	if errors.Is(err, errOrderNotFound) {
		return nil, orderNotFound()
	}
	if err != nil {
		return nil, err
	}
	return api.GetOrderChatParticipants200JSONResponse{
		ClientId: participants.ClientID, EngineerId: participants.EngineerID,
	}, nil
}

// CreateOrder выбирает форму заявки по доверенной роли из Gateway.
func (h handler) CreateOrder(ctx context.Context, req api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	// OpenAPI принимает обе формы, а роль из Gateway определяет сценарий.
	if user.Role == httpx.RoleClient {
		return h.createServiceBooking(ctx, user.ID, req.Body)
	}
	if _, err := httpx.EngineerIdentity(ctx); err != nil {
		return nil, err
	}
	body, err := req.Body.AsCreateEngineerOrderRequest()
	// Декодирование варианта oneOf само по себе не проверяет обязательные поля.
	if err != nil || body.ClientId == uuid.Nil {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed",
			Message: "engineer order requires a client ID", Fields: map[string]string{"client_id": "required"},
		}
	}
	id, err := h.createEngineerOrder(ctx, body, user.ID)
	if err != nil {
		return nil, intakeError(err)
	}
	return api.CreateOrder201JSONResponse{OrderId: id}, nil
}

// AddOrderItem получает цену из Inventory до записи. Репозиторий проверяет статус при вставке.
func (h handler) AddOrderItem(ctx context.Context, req api.AddOrderItemRequestObject) (api.AddOrderItemResponseObject, error) {
	engineer, err := httpx.EngineerIdentity(ctx)
	if err != nil {
		return nil, err
	}
	product, err := h.findItem(ctx, req.Body.ItemId)
	if err != nil {
		return nil, err
	}

	item := orderItem{itemID: req.Body.ItemId, quantity: req.Body.Quantity, name: &product.name, price: product.price}
	item.id, err = h.repo.addItem(ctx, req.Id, engineer.ID, item)
	if err != nil {
		return nil, transitionError(err, "order_not_editable", "order must be active")
	}
	return itemResponse(item), nil
}

func (h handler) findItem(ctx context.Context, itemID uuid.UUID) (catalogEntry, error) {
	product, err := h.inventory.lookup(ctx, itemID)
	if errors.Is(err, errCatalogEntryNotFound) {
		return catalogEntry{}, &httpx.Error{Status: http.StatusNotFound, Code: "item_not_found", Message: "item not found"}
	}
	if err != nil {
		h.logger.WarnContext(ctx, "could not get inventory price", "err", err)
		return catalogEntry{}, &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "inventory unavailable"}
	}

	return product, nil
}

func orderNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "order_not_found", Message: "order not found"}
}

// GetOrder возвращает карточку владельцу, ответственному и исполнителям заказа.
func (h handler) GetOrder(ctx context.Context, req api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	if user.Role == httpx.RoleClient {
		return h.readOrder(ctx, req.Id, orderRead{clientID: user.ID})
	}
	if user.Role == directorRole {
		return h.readOrder(ctx, req.Id, orderRead{includeEngineers: true})
	}
	if _, err := httpx.EngineerIdentity(ctx); err != nil {
		return nil, err
	}
	return h.readOrder(ctx, req.Id, orderRead{engineerID: user.ID, includeEngineers: true})
}

func (h handler) readOrder(ctx context.Context, id uuid.UUID, read orderRead) (orderJSON, error) {
	// Историческая карточка не расширяет доступ инженера к справкам.
	read.includeCertificateState = read.historyAnchorID == uuid.Nil
	snapshot, err := h.repo.get(ctx, id, read)
	if errors.Is(err, errOrderNotFound) {
		return orderJSON{}, orderNotFound()
	}
	if err != nil {
		return orderJSON{}, err
	}
	response := orderResponse(snapshot.header, snapshot.items, snapshot.services)
	response.CertificateStatus = snapshot.certificateStatus
	if read.includeEngineers {
		engineers := h.describeEngineers(ctx, snapshot.engineerIDs)
		response.Engineers = &engineers
	}
	if read.engineerID != uuid.Nil {
		canEdit := read.historyAnchorID == uuid.Nil && isResponsible(snapshot.header.EngineerID, read.engineerID)
		response.CanEdit = &canEdit
	}
	return response, nil
}
