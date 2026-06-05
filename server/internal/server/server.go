// Package server is the composition root: it builds the HTTP server from
// configuration and dependencies and runs it with graceful shutdown.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/config"
	"github.com/opentdm/opentdm/server/internal/httpapi"
)

// Server wraps an *http.Server with lifecycle management.
type Server struct {
	httpSrv *http.Server
	logger  *slog.Logger
}

// New builds a Server from config, a logger, the service, an optional web UI
// handler, and the readiness checks to expose at /readyz.
func New(cfg *config.Config, logger *slog.Logger, svc *app.Service, secureCookies bool, web http.Handler, checks ...httpapi.ReadyCheck) *Server {
	handler := httpapi.NewRouter(httpapi.Options{
		Logger:        logger,
		ReadyChecks:   checks,
		Service:       svc,
		SecureCookies: secureCookies,
		MaxBlobBytes:  cfg.MaxBlobBytes,
		WebHandler:    web,
	})
	return &Server{
		httpSrv: &http.Server{
			Addr:              cfg.Addr(),
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
		logger: logger,
	}
}

// Run starts serving and blocks until ctx is cancelled, then drains in-flight
// requests within a timeout. Returns nil on a clean shutdown.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("http_listen", slog.String("addr", s.httpSrv.Addr))
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("http_shutdown_begin")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := s.httpSrv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		s.logger.Info("http_shutdown_complete")
		return nil
	}
}
