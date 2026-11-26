package internal

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestChatUpgradeUsesJWTDeadlineAndRemovesCredential(t *testing.T) {
	secret := []byte("chat-upgrade-test-key")
	expiry := time.Now().Add(time.Minute).Truncate(time.Second)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		Role: roleClient,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "chat-client",
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://sto.localhost:3001"+chatEventsPath, nil)
	request.Header.Set("Sec-WebSocket-Protocol", chatProtocol+", "+signed)
	request.Header.Set("Origin", "http://sto.localhost:3001")
	chatRoute, _ := findRoute(chatEventsPath)
	authorized, cancel, err := authorizeChatEvents(request, chatRoute, secret)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	deadline, ok := authorized.Context().Deadline()
	if !ok || !deadline.Equal(expiry) {
		t.Fatalf("connection deadline = %s, want %s", deadline, expiry)
	}
	pr := &httputil.ProxyRequest{In: authorized, Out: authorized.Clone(authorized.Context())}
	pr.Out.Header.Set("Connection", "Upgrade")
	pr.Out.Header.Set("Upgrade", "websocket")
	rewriteRequest(&url.URL{Scheme: "http", Host: "chat:8080"})(pr)
	if got := pr.Out.Header.Get("Sec-WebSocket-Protocol"); got != chatProtocol {
		t.Fatal("upstream must receive only the protocol name")
	}
	if pr.Out.Host != request.Host || pr.Out.Header.Get("Origin") != request.Header.Get("Origin") {
		t.Fatal("upstream must receive public Host and Origin")
	}
	if pr.Out.Header.Get("Upgrade") != "websocket" || pr.Out.Header.Get("Connection") != "Upgrade" {
		t.Fatal("proxy rewrite must preserve the WebSocket upgrade")
	}
}
