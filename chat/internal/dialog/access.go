package dialog

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chat/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	directorRole         = "director"
	orderDialog          = "order"
	directorEngineerChat = "director_engineer"
	supportDialog        = "support"
)

type access int

const (
	noAccess access = iota
	readOnly
	readWrite
)

func (h handler) requireDialog(ctx context.Context, id uuid.UUID, required access) (db.Dialog, httpx.Identity, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return db.Dialog{}, user, httpx.Unauthorized()
	}
	current, err := h.queries.GetParticipantDialog(ctx, db.GetParticipantDialogParams{DialogID: id, UserID: user.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return current, user, dialogNotFound()
	}
	if err != nil {
		return current, user, err
	}
	permission, err := h.dialogAccess(ctx, current, user)
	if err != nil {
		return current, user, err
	}
	if permission == noAccess {
		return current, user, dialogNotFound()
	}
	if permission < required {
		return current, user, &httpx.Error{Status: http.StatusForbidden, Code: "dialog_read_only", Message: "dialog is read only"}
	}
	return current, user, nil
}

func (h handler) dialogAccess(ctx context.Context, current db.Dialog, user httpx.Identity) (access, error) {
	if !matchesParticipant(current, user) {
		return noAccess, nil
	}
	if current.Kind != orderDialog {
		return readWrite, nil
	}
	// Участие в сохранённом диалоге даёт право проверить заказ в Orders.
	// Только актуальное назначение разрешает обеим сторонам писать.
	participants, err := h.directory.orderParticipants(ctx, *current.OrderID)
	var apiErr *httpx.Error
	if errors.As(err, &apiErr) && apiErr.Code == "order_not_found" {
		return noAccess, nil
	}
	if err != nil {
		return noAccess, err
	}
	return orderDialogAccess(current, user, participants), nil
}

func matchesParticipant(current db.Dialog, user httpx.Identity) bool {
	var firstRole, secondRole string
	switch current.Kind {
	case orderDialog:
		firstRole, secondRole = httpx.RoleClient, httpx.RoleEngineer
	case directorEngineerChat:
		firstRole, secondRole = directorRole, httpx.RoleEngineer
	case supportDialog:
		firstRole, secondRole = httpx.RoleClient, directorRole
	default:
		return false
	}
	return user.ID == current.Participant1ID && user.Role == firstRole ||
		user.ID == current.Participant2ID && user.Role == secondRole
}

func orderDialogAccess(current db.Dialog, user httpx.Identity, participants orderParticipants) access {
	if participants.ClientID != current.Participant1ID {
		return noAccess
	}
	if participants.EngineerID != nil && *participants.EngineerID == current.Participant2ID {
		return readWrite
	}
	if user.Role == httpx.RoleClient {
		return readOnly
	}
	return noAccess
}

func dialogNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "dialog_not_found", Message: "dialog not found"}
}

func orderNotFound() *httpx.Error {
	return &httpx.Error{Status: http.StatusNotFound, Code: "order_not_found", Message: "order not found"}
}
