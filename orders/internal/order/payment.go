package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/saga"
)

// PayOrder подтверждает оплату заказа владельцем.
func (h handler) PayOrder(ctx context.Context, req api.PayOrderRequestObject) (api.PayOrderResponseObject, error) {
	identity, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	paidAt, err := h.payments.ConfirmPayment(ctx, req.Id, identity.ID)
	if errors.Is(err, saga.ErrNotActive) {
		return nil, &httpx.Error{Status: http.StatusConflict, Code: "order_not_active", Message: "order unavailable for payment"}
	}
	if err != nil {
		return nil, err
	}
	return api.PayOrder200JSONResponse{Status: api.MarkOrderPaidResponseStatusPaid, PaidAt: paidAt}, nil
}

// RetryOrderStock начинает новую попытку после отказа Inventory.
func (h handler) RetryOrderStock(ctx context.Context, req api.RetryOrderStockRequestObject) (api.RetryOrderStockResponseObject, error) {
	identity, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	if identity.Role != "engineer" {
		return nil, &httpx.Error{Status: http.StatusForbidden, Code: "forbidden", Message: "forbidden"}
	}
	err := h.payments.RetryStock(ctx, req.Id)
	if errors.Is(err, saga.ErrStockRetryUnavailable) {
		return nil, &httpx.Error{Status: http.StatusConflict, Code: "stock_retry_unavailable", Message: "stock deduction cannot be retried"}
	}
	if err != nil {
		return nil, err
	}
	return api.RetryOrderStock202Response{}, nil
}
