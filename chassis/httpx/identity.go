package httpx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

// Заголовки внутренней идентичности выставляет Gateway после проверки токена (ADR-037).
const (
	UserIDHeader   = "X-User-Id"
	UserRoleHeader = "X-User-Role"
)

const RoleClient = "client"

type Identity struct {
	ID   uuid.UUID
	Role string
}

type identityKey struct{}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

func Unauthorized() *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "unauthorized"}
}

// RequireIdentity reads the trusted headers set by Gateway.
func RequireIdentity(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.Header.Get(UserIDHeader))
		role := r.Header.Get(UserRoleHeader)
		if err != nil || id == uuid.Nil || role == "" {
			logger.WarnContext(r.Context(), "missing or invalid request identity")
			WriteError(w, logger, Unauthorized())
			return
		}
		ctx := context.WithValue(r.Context(), identityKey{}, Identity{ID: id, Role: role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
