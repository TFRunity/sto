package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func Run(ctx context.Context, addr string, app http.Handler, logger *slog.Logger) error {
	srv := newServer(addr, withHealthz(app))
	serverErr := startInBackground(srv)
	logger.Info("http server starting", "addr", addr)
	if err := waitForStopOrError(ctx, serverErr); err != nil {
		return err
	}
	logger.Info("http server stopping")
	return shutdownGracefully(srv)
}

func withHealthz(app http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", app)
	return mux
}

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func startInBackground(srv *http.Server) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	return errCh
}

func waitForStopOrError(ctx context.Context, serverErr <-chan error) error {
	select {
	case <-ctx.Done():
		return nil
	case err := <-serverErr:
		return err
	}
}

func shutdownGracefully(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
