package dialog

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chat/internal/db"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

func TestEventDeliveryRechecksOrderAssignment(t *testing.T) {
	clientID, previousEngineer, responsible := uuid.New(), uuid.New(), uuid.New()
	checked := make(chan struct{}, 1)
	orders := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Назначение изменилось, пока событие ожидало доставки.
		_ = json.NewEncoder(w).Encode(orderParticipants{ClientID: clientID, EngineerID: &responsible})
		checked <- struct{}{}
	}))
	defer orders.Close()
	ordersURL, err := url.Parse(orders.URL)
	if err != nil {
		t.Fatal(err)
	}

	logger := slog.Default()
	h := handler{
		events:    newEventHub(),
		directory: directory{orders: *ordersURL, http: orders.Client(), logger: logger},
	}
	server := httptest.NewServer(httpx.RequireIdentity(http.HandlerFunc(h.serveEvents), logger))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ws, response, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), &websocket.DialOptions{
		Subprotocols: []string{chatProtocol},
		HTTPHeader: http.Header{
			http.CanonicalHeaderKey(httpx.UserIDHeader):   {previousEngineer.String()},
			http.CanonicalHeaderKey(httpx.UserRoleHeader): {httpx.RoleEngineer},
		},
	})
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ws.CloseNow() }()

	orderID := uuid.New()
	h.events.publish(db.Dialog{
		ID: uuid.New(), Kind: orderDialog, OrderID: &orderID,
		Participant1ID: clientID, Participant2ID: previousEngineer,
	}, "message")
	staff := db.Dialog{
		ID: uuid.New(), Kind: directorEngineerChat,
		Participant1ID: uuid.New(), Participant2ID: previousEngineer,
	}
	h.events.publish(staff, "message")

	// FIFO-очередь сначала проверяет старый заказ, затем пропускает
	// разрешённое событие. Первым в сети должен оказаться диалог с директором.
	var event chatEvent
	if err := wsjson.Read(ctx, ws, &event); err != nil {
		t.Fatal(err)
	}
	if event.DialogID != staff.ID {
		t.Fatalf("received revoked event: %+v", event)
	}
	select {
	case <-checked:
	case <-ctx.Done():
		t.Fatal("Chat did not check the current assignment")
	}
}
