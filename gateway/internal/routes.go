package internal

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

const (
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
	{Prefix: "/orders", Upstream: "orders", Roles: []string{roleEngineer}},
}

//-------------------------------START Outer Func zone-----------------------------

func NewRouter(secret []byte, proxies map[string]http.Handler, logger *slog.Logger) http.Handler {
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, secret, proxies, logger)
	})
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", app))
	mux.Handle("/", app)
	return mux
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

	var identity Identity
	if !route.Public {
		var err error
		identity, err = Authenticate(r.Header.Get("Authorization"), secret)
		if err != nil {
			httpx.WriteError(w, logger, ErrUnauthorized)
			return
		}
	}

	route, decision := decideRequest(r, route, identity)
	if !handleDecision(w, logger, decision) {
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

	if !route.Public {
		r = r.WithContext(WithIdentity(r.Context(), identity))
	}
	proxy.ServeHTTP(w, r)
}

func decideRequest(_ *http.Request, route Route, identity Identity) (Route, Decision) {
	if route.Public || slices.Contains(route.Roles, identity.Role) {
		return route, Allow
	}
	return route, Forbidden
}

func handleDecision(w http.ResponseWriter, logger *slog.Logger, decision Decision) bool {
	if decision == Allow {
		return true
	}
	httpx.WriteError(w, logger, &httpx.Error{
		Status:  http.StatusForbidden,
		Code:    "forbidden",
		Message: "forbidden",
	})
	return false
}
