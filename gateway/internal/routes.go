package internal

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

const (
	roleClient   = "client"
	roleEngineer = "engineer"
)

type Decision int

const (
	Forbidden Decision = iota
	Unauthorized
	Allow
)

type Route struct {
	Prefix   string
	Upstream string
	Public   bool
	Roles    []string
}

var routes = []Route{
	{Prefix: "/register", Upstream: "users", Public: true},
	{Prefix: "/login", Upstream: "users", Public: true},
	{Prefix: "/orders", Upstream: "orders", Roles: []string{roleEngineer, roleClient}},
}

//-------------------------------START Outer Func zone-----------------------------

func NewRouter(secret []byte, proxies map[string]http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, secret, proxies, logger)
	})
}

//-------------------------------END Outer Func zone-------------------------------

func findRoute(path string) (Route, bool) {
	for _, route := range routes {
		if matchesPrefix(path, route.Prefix) {
			return route, true
		}
	}
	return Route{}, false
}

func matchesPrefix(path string, prefix string) bool {
	rest, found := strings.CutPrefix(path, prefix)
	return found && (rest == "" || rest[0] == '/')
}

// Управляет обработкой входящего запроса.
func handleRequest(w http.ResponseWriter, r *http.Request, secret []byte, proxies map[string]http.Handler, logger *slog.Logger) {
	route, found := findRoute(r.URL.Path)
	if !found {
		httpx.WriteError(w, logger, &httpx.Error{
			Status:  http.StatusForbidden,
			Code:    "no_such_route",
			Message: "no such route",
		})
		return
	}

	identity, hasIdentity := authenticateRequest(r, route, secret)
	if !hasIdentity {
		httpx.WriteError(w, logger, &httpx.Error{
			Status:  http.StatusBadRequest,
			Code:    "no_such_permission_role",
			Message: "no such permission role",
		})
		return
	}

	route, decision := decideRequest(r, route, identity)

	if !handleDecision(w, logger, decision) {
		httpx.WriteError(w, logger, &httpx.Error{
			Status:  http.StatusBadRequest,
			Code:    "_not_configured",
			Message: "upstream not configured",
		})
		return
	}

	proxy, found := proxies[route.Upstream]
	if !found {
		httpx.WriteError(w, logger, &httpx.Error{
			Status:  http.StatusBadGateway,
			Code:    "upstream_not_configured",
			Message: "upstream not configured",
		})
		return
	}

	proxyRequest(w, r, proxy, identity)
}

// Подтверждает личность и наличие подходящей роли
func authenticateRequest(r *http.Request, route Route, secret []byte) (Identity, bool) {
	authHeader := r.Header.Get("Authorization")
	identity, err := Authenticate(authHeader, secret)
	if err != nil {
		return Identity{}, false
	}
	if route.Public {
		return identity, true
	}
	if !slices.Contains(route.Roles, identity.Role) {
		return Identity{}, false
	}
	return identity, true
}

func decideRequest(r *http.Request, route Route, identity Identity) (Route, Decision) {
	// Сюда можно будет вписать особенные правила, всякие разные. Пока что
	// Просто будем возвращать route, Allow.
	// Например для тестировщиков, (в дальнейших сценариях понадобится)
	return route, Allow
}

// Усложнить после усложенения decideRequest
func handleDecision(w http.ResponseWriter, logger *slog.Logger, decision Decision) bool {
	if decision != Allow {
		return false
	}
	return true
}

// По умолчанию если мы используем с пустым identity - значит аноним сидит
func proxyRequest(w http.ResponseWriter, r *http.Request, proxy http.Handler, identity Identity) {
}
