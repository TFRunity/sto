package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
	"github.com/AlexeBabanov/sto/orders/internal/saga"
)

func (h handler) PayOrder(ctx context.Context, req api.PayOrderRequestObject) (api.PayOrderResponseObject, error) {
	identity, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	err := h.payments.StartPayment(ctx, req.Id, identity.ID)
	if errors.Is(err, saga.ErrNotActive) {
		return nil, &httpx.Error{Status: http.StatusConflict, Code: "order_not_active", Message: "order unavailable for payment"}
	}
	if err != nil {
		return nil, err
	}
	return api.PayOrder202Response{}, nil
}
