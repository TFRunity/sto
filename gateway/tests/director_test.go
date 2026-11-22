package tests

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/AlexeBabanov/sto/gateway/internal"
)

func TestDashboardCancelsSiblingWhenSourceFails(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	users := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer users.Close()
	orders := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-started:
			w.WriteHeader(http.StatusServiceUnavailable)
		case <-r.Context().Done():
		}
	}))
	defer orders.Close()
	usersURL, err := url.Parse(users.URL)
	if err != nil {
		t.Fatal(err)
	}
	ordersURL, err := url.Parse(orders.URL)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := internal.NewDirector(*ordersURL, *usersURL, url.URL{}, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/dashboard", nil)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d: %s", response.Code, response.Body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "upstream_error" {
		t.Fatalf("expected upstream_error, got %q", body.Error.Code)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("the parallel Users request was not cancelled")
	}
}
