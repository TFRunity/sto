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
	roleClient   = "client"
)

type route struct {
	prefix   string
	upstream string
	public   bool
	roles    []string
}

var routes = []route{
	{prefix: "/register", upstream: "users", public: true},
	{prefix: "/login", upstream: "users", public: true},
	// Запись позиций и создание заказа доступны инженеру. Исключения проверяются по методу.
	{prefix: "/orders", upstream: "orders", roles: []string{roleEngineer}},
	{prefix: "/vehicles", upstream: "vehicle"},
}

func NewRouter(secret []byte, proxies map[string]http.Handler, logger *slog.Logger) http.Handler {
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, secret, proxies, logger)
	})
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", app))
	mux.Handle("/", app)
	return mux
}

func findRoute(path string) (route, bool) {
	for _, candidate := range routes {
		if matchesPrefix(path, candidate.prefix) {
			return candidate, true
		}
	}
	return route{}, false
}

func matchesPrefix(path, prefix string) bool {
	rest, found := strings.CutPrefix(path, prefix)
	return found && (rest == "" || rest[0] == '/')
}

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

	request, err := authorizeRequest(r, route, secret)
	if err != nil {
		httpx.WriteError(w, logger, err)
		return
	}
	forwardRequest(w, request, proxies[route.upstream], logger)
}

func authorizeRequest(r *http.Request, route route, secret []byte) (*http.Request, error) {
	if route.public {
		return r, nil
	}

	identity, err := Authenticate(r.Header.Get("Authorization"), secret)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if err := decideRequest(r, route, identity); err != nil {
		return nil, err
	}
	return r.WithContext(WithIdentity(r.Context(), identity)), nil
}

func decideRequest(r *http.Request, route route, identity Identity) error {
	roles := route.roles
	switch route.prefix {
	case "/orders":
		roles = orderRoles(r, roles)
	case "/vehicles":
		roles = vehicleRoles(r)
	}
	if slices.Contains(roles, identity.Role) {
		return nil
	}
	return &httpx.Error{
		Status:  http.StatusForbidden,
		Code:    "forbidden",
		Message: "forbidden",
	}
}

func vehicleRoles(r *http.Request) []string {
	if r.Method == http.MethodPost && r.URL.Path == "/vehicles" {
		return []string{roleClient}
	}
	path, found := strings.CutPrefix(r.URL.Path, "/vehicles/")
	if !found {
		return nil
	}
	id, action, hasAction := strings.Cut(path, "/")
	if id == "" {
		return nil
	}
	resource, photoID, hasPhotoID := strings.Cut(action, "/")
	galleryPhoto := resource == "photos" && hasPhotoID && photoID != "" && !strings.Contains(photoID, "/")
	switch {
	case r.Method == http.MethodGet && (!hasAction || action == "cover_photo" || galleryPhoto):
		return []string{roleClient, roleEngineer}
	case action == "cover_photo" && r.Method == http.MethodPut:
		return []string{roleClient}
	case action == "photos" && r.Method == http.MethodPost:
		return []string{roleEngineer}
	}
	return nil
}

func forwardRequest(w http.ResponseWriter, r *http.Request, proxy http.Handler, logger *slog.Logger) {
	if proxy == nil {
		httpx.WriteError(w, logger, &httpx.Error{
			Status:  http.StatusBadGateway,
			Code:    "upstream_not_configured",
			Message: "upstream not configured",
		})
		return
	}
	proxy.ServeHTTP(w, r)
}

func orderRoles(r *http.Request, fallback []string) []string {
	if r.Method == http.MethodPost && r.URL.Path == "/orders" {
		return []string{roleEngineer, roleClient}
	}
	if r.Method == http.MethodGet && r.URL.Path == "/orders" && r.URL.Query().Has("client_id") {
		return []string{roleClient}
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/orders/"), "/")
	if len(parts) == 2 && parts[1] == "pay" && r.Method == http.MethodPost {
		return []string{roleClient}
	}
	if len(parts) == 2 && parts[1] == "certificate" && r.Method == http.MethodGet {
		return []string{roleClient}
	}
	if len(parts) == 1 && parts[0] != "/orders" && parts[0] != "" && r.Method == http.MethodGet {
		return []string{roleEngineer, roleClient}
	}
	return fallback
}
