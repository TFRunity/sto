package internal

import (
	"errors"
	"net/http"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

// Ошибки Users — объявлены только здесь (ADR-032: маппинг в HTTP — в одном месте).
// Сервис и репозиторий возвращают их, httpx.WriteError отвечает их статусом и кодом.
// Значения общие на все запросы: не изменять.
var (
	ErrPhoneTaken = &httpx.Error{Status: http.StatusConflict,
		Code: "phone_taken", Message: "phone already taken"}

	ErrInvalidCredentials = &httpx.Error{Status: http.StatusUnauthorized,
		Code: "invalid_credentials", Message: "invalid phone or password"}

	ErrPasswordTooLong = &httpx.Error{Status: http.StatusBadRequest,
		Code: "validation_failed", Message: "password too long",
		Fields: map[string]string{"password": "too_long"}}

	// Внутренняя: клиенту не уходит — сервис превращает её в ErrInvalidCredentials.
	ErrUserNotFound = errors.New("user not found")
)
