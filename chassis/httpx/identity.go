package httpx

// Заголовки внутренней идентичности выставляет Gateway после проверки токена (ADR-037).
const (
	UserIDHeader   = "X-User-Id"
	UserRoleHeader = "X-User-Role"
)
