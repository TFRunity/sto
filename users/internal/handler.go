package internal

import (
	"context"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/users/internal/api"
)

// TODO(dev): HTTP.
//   - структура запроса — валидатор по спецификации (chassis/httpx);
//     Validate() — только правила вне спецификации (≤ 72 байт пароля) → ADR-041
//   - ошибки — конвертом chassis/httpx                               → ADR-032

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(ctx context.Context, req api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	userID, token, err := h.svc.Register(ctx, req.Body.Name, req.Body.Phone, req.Body.Password)
	if err != nil {
		return nil, toAPIError(err)
	}
	return api.Register201JSONResponse{UserId: userID, Token: token}, nil
}

func (h *Handler) Login(ctx context.Context, req api.RegisterRequestObject) (api.LoginResponseObject, error) {
	token, err := h.svc.Login(ctx, req.Body.Phone, req.Body.Password)
	if err != nil {
		return nil, toAPIError(err)
	}
	return api.Login200JSONResponse{Token: token}, nil
}

var apiErrors = map[error]*httpx.Error{}
