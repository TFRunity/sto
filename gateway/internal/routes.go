package internal

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

const (
	roleDirector = "director"
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
	{prefix: "/me", upstream: "users", roles: []string{roleClient, roleEngineer, roleDirector}},
	{prefix: "/certificates", upstream: "orders", roles: []string{roleClient}},
	// Запись позиций и создание заказа доступны инженеру. Исключения проверяются по методу.
	{prefix: "/orders", upstream: "orders", roles: []string{roleEngineer}},
	{prefix: "/vehicles", upstream: "vehicle"},
	{prefix: "/inventory", upstream: "inventory"},
	{prefix: "/services", upstream: "services"},
	{prefix: "/dashboard", upstream: "director", roles: []string{roleDirector}},
	{prefix: "/director/orders", upstream: "director", roles: []string{roleDirector}},
	{prefix: "/engineer/orders", upstream: "director", roles: []string{roleEngineer}},
	{prefix: "/engineer/clients", upstream: "users", roles: []string{roleEngineer}},
	{prefix: "/engineer/vehicles", upstream: "vehicle", roles: []string{roleEngineer}},
}

// NewRouter принимает внешние пути с /api и передаёт маршрутизацию по путям из OpenAPI.
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
	case "/dashboard", "/certificates", "/engineer/orders", "/engineer/clients":
		if r.Method != http.MethodGet || r.URL.Path != route.prefix {
			roles = nil
		}
	case "/director/orders":
		id := strings.TrimPrefix(r.URL.Path, route.prefix+"/")
		if r.Method != http.MethodGet || (r.URL.Path != route.prefix && (id == "" || strings.Contains(id, "/"))) {
			roles = nil
		}
	case "/me":
		if r.URL.Path != route.prefix || (r.Method != http.MethodGet && r.Method != http.MethodPatch) {
			roles = nil
		}
	case "/engineer/vehicles":
		if r.URL.Path != route.prefix || (r.Method != http.MethodGet && r.Method != http.MethodPost) {
			roles = nil
		}
	case "/vehicles":
		roles = vehicleRoles(r)
	case "/inventory", "/services":
		roles = catalogRoles(r, route.prefix)
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
	if r.Method == http.MethodGet && (r.URL.Path == "/vehicles" || r.URL.Path == "/vehicles/makes") {
		return []string{roleClient}
	}
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
	if r.URL.Path == "/orders/stats" {
		return nil
	}
	if r.Method == http.MethodPost && r.URL.Path == "/orders" {
		return []string{roleEngineer, roleClient}
	}
	if r.Method == http.MethodGet && r.URL.Path == "/orders" && r.URL.Query().Has("client_id") {
		return []string{roleClient}
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/orders/"), "/")
	if len(parts) >= 2 && parts[1] == "vehicle-history" {
		if r.Method == http.MethodGet && len(parts) <= 3 {
			return []string{roleEngineer}
		}
		return nil
	}
	if len(parts) == 2 && parts[1] == "pay" && r.Method == http.MethodPost {
		return []string{roleClient}
	}
	if len(parts) == 2 && parts[1] == "certificate" {
		if r.Method == http.MethodGet {
			return []string{roleClient, roleDirector}
		}
		return nil
	}
	if len(parts) == 1 && parts[0] != "/orders" && parts[0] != "" && r.Method == http.MethodGet {
		return []string{roleEngineer, roleClient}
	}
	return fallback
}

func catalogRoles(r *http.Request, prefix string) []string {
	if r.URL.Path == prefix {
		switch r.Method {
		case http.MethodGet:
			if prefix == "/services" {
				return []string{roleClient, roleEngineer, roleDirector}
			}
			return []string{roleEngineer, roleDirector}
		case http.MethodPost:
			return []string{roleDirector}
		}
	}
	id, found := strings.CutPrefix(r.URL.Path, prefix+"/")
	if found && id != "" && !strings.Contains(id, "/") &&
		(r.Method == http.MethodPut || r.Method == http.MethodDelete) {
		return []string{roleDirector}
	}
	return nil
}
