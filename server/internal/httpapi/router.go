package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/email"
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
	MaxBlobBytes  int64
	Mailer        email.Mailer // nil → no-op (invite links are logged)
	BaseURL       string       // absolute base for invite links; "" derives from request
	WebHandler    http.Handler // optional SPA handler for non-/api routes

	// Per-IP rate limiting for unauthenticated auth endpoints (login, bootstrap,
	// invitation accept). RPM <= 0 disables it.
	AuthRateLimitRPM   int
	AuthRateLimitBurst int
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
	r.Use(securityHeaders(opts.SecureCookies))
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
		h := NewHandlers(opts.Service, logger, opts.SecureCookies, opts.MaxBlobBytes, opts.Mailer, opts.BaseURL)
		r.Route("/api/v1", func(api chi.Router) {
			api.Use(h.loadAuth)
			api.Use(h.audit) // record successful resource mutations (no bodies)

			// Public / dual-auth endpoints.
			api.Get("/auth/setup", h.handleSetupStatus)
			api.Get("/invitations/{token}", h.handleGetInvitation)
			api.Post("/auth/logout", h.handleLogout)

			// Credential-bearing public endpoints: rate-limited per IP to blunt
			// credential stuffing and setup/invite-token guessing.
			limiter := newIPRateLimiter(opts.AuthRateLimitRPM, opts.AuthRateLimitBurst)
			limiter.startCleanup(context.Background(), 5*time.Minute, 15*time.Minute)
			api.Group(func(pub chi.Router) {
				pub.Use(limiter.Middleware)
				pub.Post("/auth/bootstrap", h.handleBootstrap)
				pub.Post("/invitations/{token}/accept", h.handleAcceptInvitation)
				pub.Post("/auth/login", h.handleLogin)
			})
			// Consumption: session OR scoped service token (checked in handler).
			api.Get("/projects/{project}/resolve", h.handleResolve)
			// Per-file consumption: resolve a single config (base → env override).
			api.Get("/projects/{project}/configs/{config}/resolve", h.handleResolveConfig)

			// Management endpoints: require a session user + CSRF.
			api.Group(func(m chi.Router) {
				m.Use(h.requireUser)
				m.Use(h.csrf)
				m.Get("/auth/me", h.handleMe)
				m.Put("/auth/me/preferences", h.handleUpdatePreferences)
				m.Get("/projects", h.handleListProjects)
				m.Post("/projects", h.handleCreateProject)
				m.Get("/projects/{project}", h.handleGetProject)
				m.Get("/projects/{project}/environments", h.handleListEnvironments)
				m.Post("/projects/{project}/environments", h.handleCreateEnvironment)
				m.Post("/projects/{project}/environments/reorder", h.handleReorderEnvironments)
				m.Patch("/projects/{project}/environments/{environment}", h.handleUpdateEnvironment)
				m.Delete("/projects/{project}/environments/{environment}", h.handleDeleteEnvironment)
				m.Get("/projects/{project}/configs", h.handleListConfigs)
				m.Post("/projects/{project}/configs", h.handleCreateConfig)
				m.Get("/projects/{project}/configs/{config}", h.handleGetConfig)
				m.Patch("/projects/{project}/configs/{config}", h.handleUpdateConfig)
				m.Delete("/projects/{project}/configs/{config}", h.handleDeleteConfig)
				m.Get("/projects/{project}/configs/{config}/items", h.handleGetItems)
				m.Put("/projects/{project}/configs/{config}/items", h.handlePutItems)
				// File/fixture content (raw body) + versioning.
				m.Get("/projects/{project}/configs/{config}/blob", h.handleGetBlob)
				m.Put("/projects/{project}/configs/{config}/blob", h.handlePutBlob)
				m.Get("/projects/{project}/configs/{config}/versions", h.handleListVersions)
				m.Get("/projects/{project}/configs/{config}/versions/{version}", h.handleGetVersion)
				m.Get("/projects/{project}/configs/{config}/diff", h.handleDiff)
				m.Post("/projects/{project}/configs/{config}/rollback", h.handleRollback)
				m.Get("/projects/{project}/tokens", h.handleListTokens)
				m.Post("/projects/{project}/tokens", h.handleCreateToken)
				m.Delete("/projects/{project}/tokens/{token}", h.handleRevokeToken)
				// Project membership (role gating inside the handlers).
				m.Get("/projects/{project}/members", h.handleListMembers)
				m.Post("/projects/{project}/members", h.handleAddMember)
				m.Patch("/projects/{project}/members/{user}", h.handleUpdateMember)
				m.Delete("/projects/{project}/members/{user}", h.handleRemoveMember)
				// Email invitations (owner-gated in the handlers).
				m.Get("/projects/{project}/invitations", h.handleListInvitations)
				m.Post("/projects/{project}/invitations", h.handleCreateInvitation)
				m.Delete("/projects/{project}/invitations/{invitation}", h.handleRevokeInvitation)
				// Project activity feed (viewer+).
				m.Get("/projects/{project}/audit", h.handleProjectAudit)

				// PAT lifecycle is session-only (a PAT cannot mint/revoke PATs).
				m.Group(func(s chi.Router) {
					s.Use(h.requireSession)
					s.Get("/pats", h.handleListPATs)
					s.Post("/pats", h.handleCreatePAT)
					s.Delete("/pats/{pat}", h.handleRevokePAT)
				})

				// Instance-admin user directory.
				m.Group(func(a chi.Router) {
					a.Use(h.requireAdmin)
					a.Get("/users", h.handleListUsers)
					a.Patch("/users/{user}", h.handleUpdateUser)
					a.Get("/audit", h.handleGlobalAudit)
				})
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
