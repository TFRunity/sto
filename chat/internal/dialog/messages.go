package dialog

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chat/internal/api"
	"github.com/AlexeBabanov/sto/chat/internal/db"
	"github.com/jackc/pgx/v5"
)

func (h handler) ListChatMessages(ctx context.Context, req api.ListChatMessagesRequestObject) (api.ListChatMessagesResponseObject, error) {
	if _, _, err := h.requireDialog(ctx, req.Id, readOnly); err != nil {
		return nil, err
	}
	rows, err := h.queries.ListDialogMessages(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	result := make(api.ListChatMessages200JSONResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, messageResponse(row))
	}
	return result, nil
}

func (h handler) SendChatMessage(ctx context.Context, req api.SendChatMessageRequestObject) (api.SendChatMessageResponseObject, error) {
	current, user, err := h.requireDialog(ctx, req.Id, readWrite)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Body.Body) == "" || strings.ContainsRune(req.Body.Body, 0) {
		return nil, &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "invalid message body",
			Fields: map[string]string{"body": "invalid"},
		}
	}
	message, err := h.queries.CreateMessage(ctx, db.CreateMessageParams{DialogID: req.Id, SenderID: user.ID, Body: req.Body.Body})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dialogNotFound()
	}
	if err != nil {
		return nil, err
	}
	h.events.publish(current, "message")
	return api.SendChatMessage201JSONResponse(messageResponse(message)), nil
}

func (h handler) MarkChatRead(ctx context.Context, req api.MarkChatReadRequestObject) (api.MarkChatReadResponseObject, error) {
	current, user, err := h.requireDialog(ctx, req.Id, readWrite)
	if err != nil {
		return nil, err
	}
	result, err := h.queries.MarkDialogReadThrough(ctx, db.MarkDialogReadThroughParams{
		DialogID: req.Id, ReaderID: user.ID, ThroughMessageID: req.Body.ThroughMessageId,
	})
	if err != nil {
		return nil, err
	}
	if !result.BoundaryExists {
		return nil, &httpx.Error{Status: http.StatusNotFound, Code: "message_not_found", Message: "message not found"}
	}
	if result.ReadChanged {
		h.events.publish(current, "read")
	}
	return api.MarkChatRead204Response{}, nil
}

func messageResponse(message db.Message) api.ChatMessage {
	return api.ChatMessage{
		Id: message.ID, DialogId: message.DialogID, SenderId: message.SenderID,
		Body: message.Body, IsRead: message.IsRead, CreatedAt: message.CreatedAt,
	}
}
