package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/google/uuid"
)

func TestWriteErrorContract(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
		field  string
	}{
		{"wrapped API error", fmt.Errorf("private context: %w", &httpx.Error{
			Status: http.StatusBadRequest, Code: "validation_failed", Message: "invalid request",
			Fields: map[string]string{"phone": "required"},
		}), http.StatusBadRequest, "validation_failed", "required"},
		{"internal error", errors.New("private context"), http.StatusInternalServerError, "internal_error", ""},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			httpx.WriteError(w, logger, tc.err)

			var body struct {
				Error struct {
					Code   string            `json:"code"`
					Fields map[string]string `json:"fields"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != tc.status || body.Error.Code != tc.code || body.Error.Fields["phone"] != tc.field {
				t.Fatalf("got HTTP %d with %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "private context") {
				t.Fatal("internal error details reached the client")
			}
		})
	}
}

func TestRequireIdentity(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	cases := []struct {
		name string
		id   string
		role string
		want int
	}{
		{"valid", id.String(), "engineer", http.StatusNoContent},
		{"missing ID", "", "engineer", http.StatusUnauthorized},
		{"malformed ID", "invalid", "engineer", http.StatusUnauthorized},
		{"zero ID", uuid.Nil.String(), "engineer", http.StatusUnauthorized},
		{"missing role", id.String(), "", http.StatusUnauthorized},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := httpx.RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				got, ok := httpx.IdentityFromContext(r.Context())
				if !ok || got.ID != id || got.Role != tc.role {
					t.Fatalf("unexpected identity: %+v, present=%t", got, ok)
				}
				w.WriteHeader(http.StatusNoContent)
			}), logger)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			req.Header.Set("X-User-Id", tc.id)
			req.Header.Set("X-User-Role", tc.role)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.want || called != (tc.want == http.StatusNoContent) {
				t.Fatalf("got HTTP %d, handler called=%t. Want HTTP %d", w.Code, called, tc.want)
			}
		})
	}
}
