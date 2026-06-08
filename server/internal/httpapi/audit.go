package httpapi

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/opentdm/opentdm/server/internal/model"
)

// auditAction maps a matched route to a semantic action. targetParam, when set,
// is the chi URL param holding the target's id.
type auditAction struct {
	action      string
	targetType  string
	targetParam string
}

// auditActions is the allowlist of mutating routes to record, keyed by
// "METHOD <RoutePattern>". Auth/self-service routes (login, logout, bootstrap,
// accept-invite, PAT lifecycle) are intentionally absent — only resource
// mutations are audited. Unmapped mutations are skipped.
var auditActions = map[string]auditAction{
	"POST /api/v1/projects":                                        {"project.created", "project", ""},
	"POST /api/v1/projects/{project}/environments":                 {"environment.created", "environment", ""},
	"POST /api/v1/projects/{project}/environments/reorder":         {"environment.reordered", "environment", ""},
	"PATCH /api/v1/projects/{project}/environments/{environment}":  {"environment.updated", "environment", "environment"},
	"DELETE /api/v1/projects/{project}/environments/{environment}": {"environment.deleted", "environment", "environment"},
	"POST /api/v1/projects/{project}/configs":                      {"config.created", "config", ""},
	"PATCH /api/v1/projects/{project}/configs/{config}":            {"config.updated", "config", "config"},
	"DELETE /api/v1/projects/{project}/configs/{config}":           {"config.archived", "config", "config"},
	"PUT /api/v1/projects/{project}/configs/{config}/items":        {"config.items.updated", "config", "config"},
	"PUT /api/v1/projects/{project}/configs/{config}/blob":         {"config.file.updated", "config", "config"},
	"POST /api/v1/projects/{project}/configs/{config}/rollback":    {"config.rolled_back", "config", "config"},
	"POST /api/v1/projects/{project}/tokens":                       {"token.created", "token", ""},
	"DELETE /api/v1/projects/{project}/tokens/{token}":             {"token.revoked", "token", "token"},
	"POST /api/v1/projects/{project}/members":                      {"member.added", "member", ""},
	"PATCH /api/v1/projects/{project}/members/{user}":              {"member.updated", "member", "user"},
	"DELETE /api/v1/projects/{project}/members/{user}":             {"member.removed", "member", "user"},
	"POST /api/v1/projects/{project}/invitations":                  {"invitation.created", "invitation", ""},
	"DELETE /api/v1/projects/{project}/invitations/{invitation}":   {"invitation.revoked", "invitation", "invitation"},
	"PATCH /api/v1/users/{user}":                                   {"user.updated", "user", "user"},
}

// audit records successful (2xx) resource mutations after the response. It never
// reads request/response bodies, so no secret values are captured. Writes are
// best-effort on a detached context and never affect the response.
func (h *Handlers) audit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			w = ww
		}
		info := &auditInfo{}
		r = r.WithContext(withAuditInfo(r.Context(), info))

		next.ServeHTTP(w, r)

		if ww.Status() < 200 || ww.Status() >= 300 {
			return
		}
		user, ok := userFrom(r.Context())
		if !ok {
			return // no user actor (e.g. public accept-invite) — not a resource mutation we record
		}
		act, ok := auditActions[r.Method+" "+chi.RouteContext(r.Context()).RoutePattern()]
		if !ok {
			return
		}
		targetID := info.TargetID
		if act.targetParam != "" {
			if v := chi.URLParam(r, act.targetParam); v != "" {
				targetID = v
			}
		}
		actorID := user.ID
		entry := model.AuditEntry{
			ProjectID:   info.ProjectID,
			ActorUserID: &actorID,
			Action:      act.action,
			TargetType:  act.targetType,
			TargetID:    targetID,
			Status:      ww.Status(),
			IP:          clientIP(r),
		}
		// Best-effort, off the request goroutine, on a detached context so a
		// client disconnect can't drop the write and a slow insert can't hold
		// the connection.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		go func() {
			defer cancel()
			if err := h.svc.RecordAudit(ctx, entry); err != nil {
				h.logger.Error("audit_write_failed", "action", act.action, "err", err)
			}
		}()
	})
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
