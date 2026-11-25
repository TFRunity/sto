package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
)

// ChangeOrderStage разрешает выдачу после подтверждения оплаты и списания.
func (h handler) ChangeOrderStage(ctx context.Context, req api.ChangeOrderStageRequestObject) (api.ChangeOrderStageResponseObject, error) {
	err := h.repo.changeStage(ctx, req.Id, string(req.Body.Stage))
	if err != nil {
		return nil, transitionError(err, "order_not_paid", "delivery requires confirmed payment and stock deduction")
	}
	return api.ChangeOrderStage200Response{}, nil
}

// CancelOrder переводит открытую заявку или активный заказ в отменённые одним условным UPDATE.
func (h handler) CancelOrder(ctx context.Context, req api.CancelOrderRequestObject) (api.CancelOrderResponseObject, error) {
	if err := h.repo.cancel(ctx, req.Id); err != nil {
		return nil, transitionError(err, "order_not_cancellable", "order cannot be cancelled")
	}
	return api.CancelOrder200Response{}, nil
}

func transitionError(err error, code, message string) error {
	switch {
	case errors.Is(err, errOrderNotFound):
		return orderNotFound()
	case errors.Is(err, errOrderConflict):
		return &httpx.Error{Status: http.StatusConflict, Code: code, Message: message}
	default:
		return err
	}
}
