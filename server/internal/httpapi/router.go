package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/opentdm/opentdm/server/internal/app"
)

// ReadyCheck reports whether a dependency is ready to serve. A nil error means
// healthy. Used by /readyz (readiness); /healthz is liveness only.
type ReadyCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// Options configures the router.
type Options struct {
	Logger        *slog.Logger
	ReadyChecks   []ReadyCheck
	Service       *app.Service // nil disables the API (Phase 0 health-only mode)
	SecureCookies bool
	WebHandler    http.Handler // optional SPA handler for non-/api routes
}

// NewRouter builds the top-level HTTP handler.
func NewRouter(opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)

	// Liveness: process is up. No dependency checks (cheap, for orchestrators).
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}, nil)
	})

	// Readiness: all dependency checks pass.
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		results := map[string]string{}
		ok := true
		for _, c := range opts.ReadyChecks {
			if err := c.Check(req.Context()); err != nil {
				ok = false
				results[c.Name] = err.Error()
			} else {
				results[c.Name] = "ok"
			}
		}
		if !ok {
			WriteJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "not_ready", "checks": results}, nil)
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"status": "ready", "checks": results}, nil)
	})

	// API surface.
	if opts.Service != nil {
		h := NewHandlers(opts.Service, logger, opts.SecureCookies)
		r.Route("/api/v1", func(api chi.Router) {
			api.Use(h.loadAuth)

			// Public / dual-auth endpoints.
			api.Get("/auth/setup", h.handleSetupStatus)
			api.Post("/auth/bootstrap", h.handleBootstrap)
			api.Post("/auth/login", h.handleLogin)
			api.Post("/auth/logout", h.handleLogout)
			// Consumption: session OR scoped service token (checked in handler).
			api.Get("/projects/{project}/resolve", h.handleResolve)

			// Management endpoints: require a session user + CSRF.
			api.Group(func(m chi.Router) {
				m.Use(h.requireUser)
				m.Use(h.csrf)
				m.Get("/auth/me", h.handleMe)
				m.Get("/projects", h.handleListProjects)
				m.Post("/projects", h.handleCreateProject)
				m.Get("/projects/{project}", h.handleGetProject)
				m.Get("/projects/{project}/environments", h.handleListEnvironments)
				m.Get("/projects/{project}/configs", h.handleListConfigs)
				m.Post("/projects/{project}/configs", h.handleCreateConfig)
				m.Get("/projects/{project}/configs/{config}", h.handleGetConfig)
				m.Get("/projects/{project}/configs/{config}/items", h.handleGetItems)
				m.Put("/projects/{project}/configs/{config}/items", h.handlePutItems)
				m.Get("/projects/{project}/tokens", h.handleListTokens)
				m.Post("/projects/{project}/tokens", h.handleCreateToken)
				m.Delete("/projects/{project}/tokens/{token}", h.handleRevokeToken)
			})
		})
	}

	// SPA fallback (the embedded web build is wired in a later phase).
	if opts.WebHandler != nil {
		r.NotFound(opts.WebHandler.ServeHTTP)
	}

	return r
}

// requestLogger logs each request with method, path, status, and latency. It
// never logs secret values or request bodies.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)
			logger.Info("http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("latency", time.Since(start)),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
