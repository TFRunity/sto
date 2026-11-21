package internal

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/gateway/internal/api"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const directorTimeout = 3 * time.Second

type director struct {
	client  *http.Client
	orders  url.URL
	users   url.URL
	vehicle url.URL
	logger  *slog.Logger
}

type personSummary struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Phone string    `json:"phone"`
}

type vehicleSummary struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
}

type orderStatsSource struct {
	Stats          api.OrderStats `json:"stats"`
	AssignedCounts []struct {
		ID           uuid.UUID `json:"id"`
		ActiveOrders int64     `json:"active_orders"`
	} `json:"assigned_counts"`
}

// NewDirector собирает HTTP-обработчик своих маршрутов Gateway. Проверку роли выполняет внешний роутер.
func NewDirector(orders, users, vehicle url.URL, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, err
	}
	h := director{
		client: &http.Client{Timeout: directorTimeout + time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		orders: orders, users: users, vehicle: vehicle, logger: logger,
	}
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions(httpx.NewStrictErrors(logger)))
	return httpx.Validator(spec, logger)(api.Handler(strict)), nil
}

// GetDashboard параллельно читает метрики и список инженеров под одним дедлайном. При сбое ответ целиком отклоняется.
func (h director) GetDashboard(ctx context.Context, _ api.GetDashboardRequestObject) (api.GetDashboardResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, directorTimeout)
	defer cancel()
	group, work := errgroup.WithContext(ctx)
	var orders orderStatsSource
	var people []personSummary
	group.Go(func() error {
		var err error
		orders, _, err = httpx.GetJSON[orderStatsSource](work, h.client, h.orders, "/orders/stats", nil)
		return err
	})
	group.Go(func() error {
		var err error
		people, _, err = httpx.GetJSON[[]personSummary](work, h.client, h.users, "/users/engineers-load", nil)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, h.sourceError(ctx, err)
	}
	counts := make(map[uuid.UUID]int64, len(orders.AssignedCounts))
	for _, count := range orders.AssignedCounts {
		counts[count.ID] = count.ActiveOrders
	}
	engineers := make([]api.EngineerLoad, 0, len(people))
	for _, person := range people {
		engineers = append(engineers, api.EngineerLoad{Id: person.ID, Name: person.Name, ActiveOrders: counts[person.ID]})
	}
	return api.GetDashboard200JSONResponse{Orders: orders.Stats, Engineers: engineers}, nil
}

// ListDirectorOrders использует один дедлайн для поиска ID, чтения страницы и загрузки подписей.
func (h director) ListDirectorOrders(ctx context.Context, req api.ListDirectorOrdersRequestObject) (api.ListDirectorOrdersResponseObject, error) {
	ctx, cancel := context.WithTimeout(ctx, directorTimeout)
	defer cancel()
	page := int32(1)
	if req.Params.Page != nil {
		page = *req.Params.Page
	}
	params, noMatches, err := h.orderPageParams(ctx, req.Params, page)
	if err != nil {
		return nil, err
	}
	if noMatches {
		return api.ListDirectorOrders200JSONResponse{Orders: []api.DirectorOrder{}, Page: page}, nil
	}
	result, _, err := httpx.GetJSON[api.DirectorOrderPage](ctx, h.client, h.orders, "/internal/orders", params)
	if err != nil {
		return nil, h.sourceError(ctx, err)
	}
	if err := h.enrich(ctx, result.Orders); err != nil {
		return nil, h.sourceError(ctx, err)
	}
	return api.ListDirectorOrders200JSONResponse(result), nil
}

// orderPageParams translates text to IDs before Orders applies pagination.
func (h director) orderPageParams(ctx context.Context, input api.ListDirectorOrdersParams, page int32) (url.Values, bool, error) {
	params := url.Values{"page": {strconv.FormatInt(int64(page), 10)}}
	if input.Status != nil {
		params.Set("status", string(*input.Status))
	}
	if input.Q == nil {
		return params, false, nil
	}
	query := strings.TrimSpace(*input.Q)
	if utf8.RuneCountInString(query) < 2 {
		return nil, false, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "search requires at least two characters",
			Fields: map[string]string{"q": "minLength"},
		}
	}
	clients, vehicles, err := h.findIDs(ctx, query)
	if err != nil {
		return nil, false, h.sourceError(ctx, err)
	}
	for _, id := range clients {
		params.Add("client_ids", id.String())
	}
	for _, id := range vehicles {
		params.Add("vehicle_ids", id.String())
	}
	return params, len(clients) == 0 && len(vehicles) == 0, nil
}

// findIDs ищет клиентов и ТС одновременно. Любой источник может ограничить ширину запроса кодом 422.
func (h director) findIDs(ctx context.Context, query string) ([]uuid.UUID, []uuid.UUID, error) {
	params := url.Values{"q": {query}}
	group, work := errgroup.WithContext(ctx)
	var clients, vehicles []uuid.UUID
	group.Go(func() error {
		result, status, err := httpx.GetJSON[[]uuid.UUID](work, h.client, h.users, "/users/search", params)
		if status == http.StatusUnprocessableEntity {
			return httpx.SearchTooBroad()
		}
		clients = result
		return err
	})
	group.Go(func() error {
		result, status, err := httpx.GetJSON[[]uuid.UUID](work, h.client, h.vehicle, "/internal/vehicles/search", params)
		if status == http.StatusUnprocessableEntity {
			return httpx.SearchTooBroad()
		}
		vehicles = result
		return err
	})
	err := group.Wait()
	return clients, vehicles, err
}

func (h director) enrich(ctx context.Context, orders []api.DirectorOrder) error {
	if len(orders) == 0 {
		return nil
	}
	peopleIDs, vehicleIDs := orderDirectoryIDs(orders)
	group, work := errgroup.WithContext(ctx)
	var people []personSummary
	var vehicles []vehicleSummary
	group.Go(func() error {
		var err error
		people, _, err = httpx.GetJSON[[]personSummary](work, h.client, h.users, "/users", idsParams(peopleIDs))
		return err
	})
	group.Go(func() error {
		var err error
		vehicles, _, err = httpx.GetJSON[[]vehicleSummary](work, h.client, h.vehicle, "/internal/vehicles", idsParams(vehicleIDs))
		return err
	})
	if err := group.Wait(); err != nil {
		return err
	}
	applyDirectoryData(orders, people, vehicles)
	return nil
}

// orderDirectoryIDs keeps each source lookup to one batch across the page.
func orderDirectoryIDs(orders []api.DirectorOrder) (map[uuid.UUID]struct{}, map[uuid.UUID]struct{}) {
	peopleIDs := make(map[uuid.UUID]struct{})
	vehicleIDs := make(map[uuid.UUID]struct{})
	for _, row := range orders {
		peopleIDs[row.ClientId] = struct{}{}
		if row.EngineerId != nil {
			peopleIDs[*row.EngineerId] = struct{}{}
		}
		vehicleIDs[row.VehicleId] = struct{}{}
	}
	return peopleIDs, vehicleIDs
}

// applyDirectoryData preserves orders with missing references and fills the names it found.
func applyDirectoryData(orders []api.DirectorOrder, people []personSummary, vehicles []vehicleSummary) {
	byPerson := make(map[uuid.UUID]personSummary, len(people))
	for _, person := range people {
		byPerson[person.ID] = person
	}
	byVehicle := make(map[uuid.UUID]string, len(vehicles))
	for _, vehicle := range vehicles {
		byVehicle[vehicle.ID] = vehicle.Number
	}
	for i := range orders {
		row := &orders[i]
		if person, ok := byPerson[row.ClientId]; ok {
			row.ClientName, row.ClientPhone = &person.Name, &person.Phone
		}
		if row.EngineerId != nil {
			if person, ok := byPerson[*row.EngineerId]; ok {
				row.EngineerName = &person.Name
			}
		}
		if number, ok := byVehicle[row.VehicleId]; ok {
			row.VehicleNumber = &number
		}
	}
}

func idsParams(ids map[uuid.UUID]struct{}) url.Values {
	params := make(url.Values)
	for id := range ids {
		params.Add("ids", id.String())
	}
	return params
}

func (h director) sourceError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &httpx.Error{Status: http.StatusGatewayTimeout, Code: "upstream_timeout", Message: "director request timed out"}
	}
	var apiError *httpx.Error
	if errors.As(err, &apiError) && apiError.Status == http.StatusUnprocessableEntity {
		return apiError
	}
	h.logger.WarnContext(ctx, "director source failed", "err", err)
	return &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "director source unavailable"}
}
