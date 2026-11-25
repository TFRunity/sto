package order

import (
	"context"
	"net/http"
	"net/url"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/google/uuid"
)

type peopleDirectory struct {
	baseURL url.URL
	http    *http.Client
}

func (d peopleDirectory) names(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	params := url.Values{}
	for _, id := range ids {
		params.Add("ids", id.String())
	}
	type person struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	people, _, err := httpx.GetJSON[[]person](ctx, d.http, d.baseURL, "/users", params)
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(people))
	for _, person := range people {
		names[person.ID] = person.Name
	}
	return names, nil
}

func (h handler) describeEngineers(ctx context.Context, ids []uuid.UUID) []api.OrderEngineer {
	result := make([]api.OrderEngineer, 0, len(ids))
	if len(ids) == 0 {
		return result
	}
	names, err := h.people.names(ctx, ids)
	if err != nil {
		// Карточка и действия инженера остаются доступны при сбое справочника Users.
		h.logger.WarnContext(ctx, "could not resolve order engineer names", "err", err)
	}
	for _, id := range ids {
		engineer := api.OrderEngineer{Id: id}
		if name := names[id]; name != "" {
			engineer.Name = &name
		}
		result = append(result, engineer)
	}
	return result
}
