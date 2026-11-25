package internal

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chassis/postgres"
	"github.com/AlexeBabanov/sto/users/internal/api"
)

// SearchClients ищет в исторических данных и сигнализирует о превышении лимита вместо усечения.
func (h *Handler) SearchClients(ctx context.Context, req api.SearchClientsRequestObject) (api.SearchClientsResponseObject, error) {
	ids, err := h.directory.q.SearchClients(ctx, postgres.ContainsPattern(strings.TrimSpace(req.Params.Q)))
	if err != nil {
		return nil, err
	}
	if len(ids) > httpx.SearchLimit {
		return nil, httpx.SearchTooBroad()
	}
	return append(api.SearchClients200JSONResponse{}, ids...), nil
}

// SearchActiveClients возвращает инженеру клиентов, доступных для прямого приёма.
func (h *Handler) SearchActiveClients(ctx context.Context, req api.SearchActiveClientsRequestObject) (api.SearchActiveClientsResponseObject, error) {
	if _, err := httpx.EngineerIdentity(ctx); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(req.Params.Q)
	if utf8.RuneCountInString(query) < 2 {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "search requires at least two characters",
			Fields: map[string]string{"q": "too_short"},
		}
	}
	rows, err := h.directory.q.SearchActiveClients(ctx, postgres.ContainsPattern(query))
	if err != nil {
		return nil, err
	}
	if len(rows) > httpx.SearchLimit {
		return nil, httpx.SearchTooBroad()
	}
	result := make(api.SearchActiveClients200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.PersonSummary{Id: row.ID, Name: row.Name, Phone: row.Phone})
	}
	return result, nil
}

// GetPeopleByIDs выдаёт краткие данные по существующим ID, включая мягко удалённых пользователей.
func (h *Handler) GetPeopleByIDs(ctx context.Context, req api.GetPeopleByIDsRequestObject) (api.GetPeopleByIDsResponseObject, error) {
	rows, err := h.directory.q.GetPeopleByIDs(ctx, req.Params.Ids)
	if err != nil {
		return nil, err
	}
	result := make(api.GetPeopleByIDs200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.PersonSummary{Id: row.ID, Name: row.Name, Phone: row.Phone})
	}
	return result, nil
}

// ListEngineersForLoad возвращает действующих инженеров. Число активных заказов Gateway берёт из Orders.
func (h *Handler) ListEngineersForLoad(ctx context.Context, _ api.ListEngineersForLoadRequestObject) (api.ListEngineersForLoadResponseObject, error) {
	rows, err := h.directory.q.ListEngineersForLoad(ctx)
	if err != nil {
		return nil, err
	}
	result := make(api.ListEngineersForLoad200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, api.PersonSummary{Id: row.ID, Name: row.Name, Phone: row.Phone})
	}
	return result, nil
}
