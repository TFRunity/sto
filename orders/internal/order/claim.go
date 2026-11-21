package order

import (
	"context"
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/orders/internal/api"
)

func (h handler) ClaimOrder(ctx context.Context, req api.ClaimOrderRequestObject) (api.ClaimOrderResponseObject, error) {
	user, ok := httpx.IdentityFromContext(ctx)
	if !ok {
		return nil, httpx.Unauthorized()
	}
	err := h.repo.claim(ctx, req.Id, user.ID)
	switch {
	case errors.Is(err, errOrderNotFound):
		return nil, orderNotFound()
	case errors.Is(err, errOrderNotClaimable):
		return nil, &httpx.Error{
			Status: http.StatusConflict, Code: "order_not_claimable", Message: "order cannot be claimed",
		}
	case err != nil:
		return nil, err
	default:
		return api.ClaimOrder200Response{}, nil
	}
}
