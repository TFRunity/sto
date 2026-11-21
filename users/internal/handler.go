package internal

import (
	"context"

	"github.com/AlexeBabanov/sto/users/internal/api"
)

// Handler — HTTP-слой Users: реализует api.StrictServerInterface (ADR-028).
// Тело запроса уже проверил валидатор (ADR-041): Body не nil, обязательные поля есть.
// Ошибки сервиса возвращаются как есть: ошибки Users — *httpx.Error (errors.go),
// их статус и код пишет httpx.StrictErrorHandler; любая другая → 500.
type Handler struct {
	svc       *Service
	directory *Repository
}

func NewHandler(svc *Service, directory *Repository) *Handler {
	return &Handler{svc: svc, directory: directory}
}

// Register — POST /register: создать клиента и сразу выдать токен.
func (h *Handler) Register(ctx context.Context, req api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	userID, token, err := h.svc.Register(ctx, req.Body.Name, req.Body.Phone, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return api.Register201JSONResponse{UserId: userID, Token: token}, nil
}

// Login — POST /login: проверить телефон и пароль, выдать токен.
func (h *Handler) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	token, err := h.svc.Login(ctx, req.Body.Phone, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return api.Login200JSONResponse{Token: token}, nil
}
