package internal

import (
	"context"
	"net/http"
	"net/http/httputil"
	"strings"
)

const (
	chatEventsPath = "/chat/events"
	chatProtocol   = "sto.chat.v1"
)

func authorizeChatEvents(r *http.Request, route route, secret []byte) (*http.Request, func(), error) {
	// Браузер передаёт JWT вторым значением Sec-WebSocket-Protocol.
	// JWT остаётся в Gateway, а Chat получает только согласованный протокол.
	protocols := strings.Join(r.Header.Values("Sec-WebSocket-Protocol"), ",")
	protocol, credential, ok := strings.Cut(protocols, ",")
	if !ok || strings.TrimSpace(protocol) != chatProtocol {
		return nil, nil, ErrUnauthorized
	}
	claims, err := verifiedClaims(strings.TrimSpace(credential), secret)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}
	identity, err := identityFrom(claims)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}
	if err := decideRequest(r, route, identity); err != nil {
		return nil, nil, err
	}
	// ReverseProxy завершает туннель при отмене контекста.
	ctx, cancel := context.WithDeadline(r.Context(), claims.ExpiresAt.Time)
	return r.WithContext(WithIdentity(ctx, identity)), cancel, nil
}

func copyChatUpgrade(pr *httputil.ProxyRequest) {
	// Chat сверяет Origin с исходным Host, включая порт.
	pr.Out.Host = pr.In.Host
	pr.Out.Header.Set("Connection", "Upgrade")
	pr.Out.Header.Set("Upgrade", "websocket")
	copyHeader(pr, "Origin")
	copyHeader(pr, "Sec-WebSocket-Key")
	copyHeader(pr, "Sec-WebSocket-Version")
	pr.Out.Header.Set("Sec-WebSocket-Protocol", chatProtocol)
}
