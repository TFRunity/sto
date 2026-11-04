// Package httpx — общий HTTP-код сервисов: конверт ошибок, идентичность, сервер.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Error — ошибка API: HTTP-статус и стабильный code для клиента (ADR-032).
// Хендлер переводит в неё доменную ошибку; любую другую ошибку WriteError отдаёт как 500.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"` // только у validation_failed
}

// Error реализует error на указателе: передавать всегда &httpx.Error{…}.
func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

// WriteJSON пишет v как JSON с указанным статусом.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // статус уже отправлен — ошибку записи вернуть некуда
}

// WriteError отвечает ошибкой в конверте {"error": {...}} (ADR-032).
//
// Как пользоваться:
//
//	httpx.WriteError(w, logger, err)
//	return // ответ уже отправлен
//
//   - нужен конкретный код для клиента (409 phone_taken) — передай
//     &httpx.Error{Status: …, Code: …, Message: …};
//   - любая другая ошибка — передай как есть: текст уйдёт только в лог,
//     клиенту — 500 internal_error.
func WriteError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		logger.Error("internal error", "err", err)
		apiErr = &Error{Status: http.StatusInternalServerError, Code: "internal_error", Message: "internal error"}
	}
	WriteJSON(w, apiErr.Status, map[string]*Error{"error": apiErr})
}
