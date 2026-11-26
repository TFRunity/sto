package dialog

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/google/uuid"
)

type directory struct {
	orders url.URL
	users  url.URL
	http   *http.Client
	logger *slog.Logger
}

type person struct {
	ID uuid.UUID `json:"id"`
}

type orderParticipants struct {
	ClientID   uuid.UUID  `json:"client_id"`
	EngineerID *uuid.UUID `json:"engineer_id"`
}

func (d directory) orderParticipants(ctx context.Context, id uuid.UUID) (orderParticipants, error) {
	path := "/internal/orders/" + id.String() + "/chat-participants"
	participants, status, err := httpx.GetJSON[orderParticipants](ctx, d.http, d.orders, path, nil)
	if status == http.StatusNotFound {
		return participants, orderNotFound()
	}
	if err != nil {
		return participants, d.upstreamError(ctx, err)
	}
	if participants.ClientID == uuid.Nil || (participants.EngineerID != nil && *participants.EngineerID == uuid.Nil) {
		return participants, d.upstreamError(ctx, fmt.Errorf("invalid chat participants for order %s", id))
	}
	return participants, nil
}

func (d directory) requireEngineer(ctx context.Context, id uuid.UUID) error {
	people, _, err := httpx.GetJSON[[]person](ctx, d.http, d.users, "/users/engineers-load", nil)
	if err != nil {
		return d.upstreamError(ctx, err)
	}
	if !slices.ContainsFunc(people, func(p person) bool { return p.ID == id }) {
		return &httpx.Error{Status: http.StatusNotFound, Code: "engineer_not_found", Message: "engineer not found"}
	}
	return nil
}

func (d directory) upstreamError(ctx context.Context, err error) *httpx.Error {
	d.logger.WarnContext(ctx, "read chat directory", "err", err)
	return &httpx.Error{Status: http.StatusBadGateway, Code: "upstream_error", Message: "chat directory unavailable"}
}
