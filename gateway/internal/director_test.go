package internal

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/gateway/internal/api"
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
	h := director{client: orders.Client(), orders: *ordersURL, users: *usersURL, logger: slog.Default()}
	response, err := h.GetDashboard(t.Context(), api.GetDashboardRequestObject{})

	var apiError *httpx.Error
	if response != nil || !errors.As(err, &apiError) || apiError.Status != http.StatusBadGateway || apiError.Code != "upstream_error" {
		t.Fatalf("expected an upstream error without a partial dashboard, got %v, %v", response, err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("the parallel Users request was not cancelled")
	}
}
