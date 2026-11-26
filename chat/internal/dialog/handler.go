package dialog

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chat/internal/api"
	"github.com/AlexeBabanov/sto/chat/internal/db"
	"github.com/google/uuid"
)

type handler struct {
	queries   *db.Queries
	directory directory
	supportID uuid.UUID
	events    *eventHub
}

func NewHTTPHandler(pool db.DBTX, ordersURL, usersURL url.URL, supportID uuid.UUID, logger *slog.Logger) (http.Handler, error) {
	spec, err := api.GetSwagger()
	if err != nil {
		return nil, err
	}
	h := handler{
		queries: db.New(pool), supportID: supportID, events: newEventHub(),
		directory: directory{orders: ordersURL, users: usersURL, http: &http.Client{Timeout: 3 * time.Second}, logger: logger},
	}
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions(httpx.NewStrictErrors(logger)))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /chat/events", h.serveEvents)
	mux.Handle("/", httpx.Validator(spec, logger)(api.Handler(strict)))
	return httpx.RequireIdentity(mux, logger), nil
}

func (h handler) OpenOrderDialog(ctx context.Context, req api.OpenOrderDialogRequestObject) (api.OpenOrderDialogResponseObject, error) {
	user, err := requireRole(ctx, httpx.RoleClient)
	if err != nil {
		return nil, err
	}
	participants, err := h.directory.orderParticipants(ctx, req.Body.OrderId)
	if err != nil {
		return nil, err
	}
	if participants.ClientID != user.ID {
		return nil, orderNotFound()
	}
	if participants.EngineerID == nil {
		return nil, &httpx.Error{Status: http.StatusConflict, Code: "order_engineer_not_assigned", Message: "order engineer not assigned"}
	}
	id, err := h.queries.OpenOrderDialog(ctx, db.OpenOrderDialogParams{
		OrderID: req.Body.OrderId, ClientID: user.ID, EngineerID: *participants.EngineerID,
	})
	return api.OpenOrderDialog200JSONResponse{DialogId: id}, err
}

func (h handler) OpenDirectorEngineerDialog(ctx context.Context, req api.OpenDirectorEngineerDialogRequestObject) (api.OpenDirectorEngineerDialogResponseObject, error) {
	user, err := requireRole(ctx, directorRole)
	if err != nil {
		return nil, err
	}
	if err := h.directory.requireEngineer(ctx, req.Body.EngineerId); err != nil {
		return nil, err
	}
	id, err := h.queries.OpenDirectorEngineerDialog(ctx, db.OpenDirectorEngineerDialogParams{
		DirectorID: user.ID, EngineerID: req.Body.EngineerId,
	})
	return api.OpenDirectorEngineerDialog200JSONResponse{
		ChatDialogOpenedJSONResponse: api.ChatDialogOpenedJSONResponse{DialogId: id},
	}, err
}

func (h handler) OpenSupportDialog(ctx context.Context, _ api.OpenSupportDialogRequestObject) (api.OpenSupportDialogResponseObject, error) {
	user, err := requireRole(ctx, httpx.RoleClient)
	if err != nil {
		return nil, err
	}
	id, err := h.queries.OpenSupportDialog(ctx, db.OpenSupportDialogParams{ClientID: user.ID, DirectorID: h.supportID})
	return api.OpenSupportDialog200JSONResponse{
		ChatDialogOpenedJSONResponse: api.ChatDialogOpenedJSONResponse{DialogId: id},
	}, err
}

func (h handler) ListChatDialogs(ctx context.Context, _ api.ListChatDialogsRequestObject) (api.ListChatDialogsResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	rows, err := h.queries.ListParticipantDialogs(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	result := make(api.ListChatDialogs200JSONResponse, 0, len(rows))
	for _, row := range rows {
		current := db.Dialog{
			ID: row.ID, Kind: row.Kind, OrderID: row.OrderID,
			Participant1ID: row.Participant1ID, Participant2ID: row.Participant2ID,
		}
		permission, err := h.dialogAccess(ctx, current, user)
		if err != nil {
			return nil, err
		}
		if permission != noAccess {
			result = append(result, dialogResponse(row, user.ID, permission))
		}
	}
	return result, nil
}

func dialogResponse(row db.ListParticipantDialogsRow, userID uuid.UUID, permission access) api.ChatDialog {
	peer := row.Participant1ID
	if peer == userID {
		peer = row.Participant2ID
	}
	unread := row.UnreadCount
	if permission == readOnly {
		// В архивном диалоге получатель уже не может менять прочитанность.
		unread = 0
	}
	return api.ChatDialog{
		Id: row.ID, Kind: api.ChatDialogKind(row.Kind), OrderId: row.OrderID, PeerId: peer,
		LastActivityAt: row.LastActivityAt, UnreadCount: unread, CanWrite: permission == readWrite,
	}
}

func requireRole(ctx context.Context, role string) (httpx.Identity, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return user, httpx.Unauthorized()
	}
	if user.Role != role {
		return user, &httpx.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "forbidden"}
	}
	return user, nil
}
