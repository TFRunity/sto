package dialog

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/AlexeBabanov/sto/chassis/httpx"
	"github.com/AlexeBabanov/sto/chat/internal/db"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

const chatProtocol = "sto.chat.v1"

type chatEvent struct {
	Type     string    `json:"type"`
	DialogID uuid.UUID `json:"dialog_id"`
}

type queuedEvent struct {
	data   chatEvent
	dialog db.Dialog
}

type eventSubscriber struct {
	queue  chan queuedEvent
	cancel context.CancelFunc
}

type eventHub struct {
	mu          sync.Mutex
	connections map[uuid.UUID]map[*eventSubscriber]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{connections: make(map[uuid.UUID]map[*eventSubscriber]struct{})}
}

func (hub *eventHub) subscribe(userID uuid.UUID, subscriber *eventSubscriber) func() {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	if hub.connections[userID] == nil {
		hub.connections[userID] = make(map[*eventSubscriber]struct{})
	}
	hub.connections[userID][subscriber] = struct{}{}
	return func() {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		delete(hub.connections[userID], subscriber)
		if len(hub.connections[userID]) == 0 {
			delete(hub.connections, userID)
		}
	}
}

func (hub *eventHub) publish(current db.Dialog, kind string) {
	event := queuedEvent{data: chatEvent{Type: kind, DialogID: current.ID}, dialog: current}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for _, userID := range []uuid.UUID{current.Participant1ID, current.Participant2ID} {
		for subscriber := range hub.connections[userID] {
			select {
			case subscriber.queue <- event:
			default:
				// Переподключение запускает HTTP-восстановление пропущенных событий.
				subscriber.cancel()
			}
		}
	}
}

func (h handler) serveEvents(w http.ResponseWriter, r *http.Request) {
	user, _ := httpx.IdentityFromContext(r.Context()) // Проверено RequireIdentity до маршрутизации.
	ctx, cancel := context.WithCancel(context.WithoutCancel(r.Context()))
	defer cancel()

	// Регистрируем соединение до ответа 101. Начальная HTTP-сверка клиента
	// покрывает более ранние изменения, а последующие уже попадут в очередь.
	subscriber := &eventSubscriber{queue: make(chan queuedEvent, 32), cancel: cancel}
	unsubscribe := h.events.subscribe(user.ID, subscriber)
	defer unsubscribe()

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{chatProtocol}})
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	h.writeEvents(conn.CloseRead(ctx), conn, user, subscriber.queue)
}

func (h handler) writeEvents(ctx context.Context, conn *websocket.Conn, user httpx.Identity, queue <-chan queuedEvent) {
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			ioCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(ioCtx)
			cancel()
			if err != nil {
				return
			}
		case event := <-queue:
			if err := h.deliverEvent(ctx, conn, user, event); err != nil {
				return
			}
		}
	}
}

func (h handler) deliverEvent(ctx context.Context, conn *websocket.Conn, user httpx.Identity, event queuedEvent) error {
	// Назначение могло измениться после HTTP-записи или пока событие ждало в очереди.
	permission, err := h.dialogAccess(ctx, event.dialog, user)
	if err != nil || permission != readWrite {
		return err
	}
	ioCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return wsjson.Write(ioCtx, conn, event.data)
}
