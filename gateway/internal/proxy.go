package internal

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/AlexeBabanov/sto/chassis/httpx"
)

type identityKey struct{}

var allowedHeaders = []string{
	"Content-Type",
	"Accept",
	"Accept-Language",
}

// WithIdentity сохраняет identity в контексте запроса.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

// IdentityFromContext получает identity из контекста.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// NewProxy создаёт ReverseProxy для указанного upstream.
func NewProxy(target *url.URL, logger *slog.Logger) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport: newTransport(),
		Rewrite:   rewriteRequest(target),
		ErrorHandler: func(
			w http.ResponseWriter,
			r *http.Request,
			err error,
		) {
			handleProxyError(w, logger, err)
		},
	}
}

// newTransport создаёт HTTP-транспорт.
func newTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 10 * time.Second
	return transport
}

// rewriteRequest возвращает функцию подготовки исходящего запроса.
func rewriteRequest(target *url.URL) func(*httputil.ProxyRequest) {
	return func(pr *httputil.ProxyRequest) {
		setTarget(pr, target)
		resetHeaders(pr)
		copyAllowedHeaders(pr)
		setIdentityHeaders(pr)
		setRequestID(pr)
	}
}

// setTarget устанавливает адрес целевого сервиса.
func setTarget(pr *httputil.ProxyRequest, target *url.URL) {
	pr.SetURL(target)
}

// resetHeaders удаляет все исходящие заголовки.
func resetHeaders(pr *httputil.ProxyRequest) {
	pr.Out.Header = make(http.Header)
}

// copyAllowedHeaders копирует только разрешённые заголовки.
func copyAllowedHeaders(pr *httputil.ProxyRequest) {
	for _, name := range allowedHeaders {
		copyHeader(pr, name)
	}
}

// copyHeader копирует все значения одного заголовка.
func copyHeader(pr *httputil.ProxyRequest, name string) {
	for _, value := range pr.In.Header.Values(name) {
		pr.Out.Header.Add(name, value)
	}
}

// setIdentityHeaders добавляет identity из контекста.
func setIdentityHeaders(pr *httputil.ProxyRequest) {
	identity, ok := IdentityFromContext(pr.In.Context())
	if !ok {
		return
	}

	pr.Out.Header.Set("X-User-Id", identity.UserID)
	pr.Out.Header.Set("X-User-Role", identity.Role)
}

// setRequestID генерирует новый Request ID.
func setRequestID(pr *httputil.ProxyRequest) {
	pr.Out.Header.Set("X-Request-Id", uuid.NewString())
}

// handleProxyError обрабатывает ошибку проксирования.
func handleProxyError(
	w http.ResponseWriter,
	logger *slog.Logger,
	err error,
) {
	apiErr := classifyProxyError(err)
	httpx.WriteError(w, logger, apiErr)
}

// classifyProxyError преобразует ошибку в API-ошибку.
func classifyProxyError(err error) *httpx.Error {
	if isTimeout(err) {
		return &httpx.Error{
			Status:  http.StatusGatewayTimeout,
			Code:    "gateway_timeout",
			Message: "upstream timeout",
		}
	}

	return &httpx.Error{
		Status:  http.StatusBadGateway,
		Code:    "bad_gateway",
		Message: "upstream unavailable",
	}
}

// isTimeout определяет, является ли ошибка таймаутом.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
