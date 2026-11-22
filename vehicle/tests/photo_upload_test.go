package tests

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	vehicle "github.com/AlexeBabanov/sto/vehicle/internal"
)

func TestLimitPhotoUpload(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/vehicles/example/photos", bytes.NewReader(make([]byte, 4*1024*1024)))
	req.ContentLength = -1
	got := httptest.NewRecorder()
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("oversized request reached handler") })
	onError := func(w http.ResponseWriter, _ *http.Request, err error) {
		var apiErr *httpx.Error
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected API error, got %v", err)
		}
		w.WriteHeader(apiErr.Status)
	}
	vehicle.LimitPhotoUpload(next, onError).ServeHTTP(got, req)
	if got.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", got.Code)
	}
}
