package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/model"
)

// Project role names (aliases of the model constants) for route gating.
const (
	roleViewer = model.RoleViewer
	roleEditor = model.RoleEditor
	roleOwner  = model.RoleOwner
)

// loadProject loads the {project} param (slug or UUID) WITHOUT an access check.
// Used by /resolve, which authorizes via token scope or session membership.
func (h *Handlers) loadProject(w http.ResponseWriter, r *http.Request) (model.Project, bool) {
	ref := chi.URLParam(r, "project")
	p, err := h.svc.GetProject(r.Context(), ref)
	if err != nil {
		if err == app.ErrNotFound {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "Project not found", "")
			return model.Project{}, false
		}
		h.writeErr(w, r, err)
		return model.Project{}, false
	}
	return p, true
}

// resolveProject loads the project and requires the caller to be at least a
// viewer member (admins bypass). It is the default gate for management reads.
func (h *Handlers) resolveProject(w http.ResponseWriter, r *http.Request) (model.Project, bool) {
	return h.resolveProjectRole(w, r, roleViewer)
}

// resolveProjectRole loads the project and enforces a minimum role: a non-member
// gets 404 (existence is hidden, GitHub-style); a member below minRole gets 403.
func (h *Handlers) resolveProjectRole(w http.ResponseWriter, r *http.Request, minRole string) (model.Project, bool) {
	p, ok := h.loadProject(w, r)
	if !ok {
		return model.Project{}, false
	}
	user, _ := userFrom(r.Context()) // management routes sit behind requireUser
	role, member, err := h.svc.ProjectRole(r.Context(), user, p.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return model.Project{}, false
	}
	if !member {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Project not found", "")
		return model.Project{}, false
	}
	if model.RoleRank(role) < model.RoleRank(minRole) {
		WriteProblem(w, r, http.StatusForbidden, "forbidden", "Insufficient project role", "")
		return model.Project{}, false
	}
	return p, true
}

// callerRole returns the caller's effective role on a project for DTOs (admins
// report "owner"). Assumes membership was already verified.
func (h *Handlers) callerRole(r *http.Request, p model.Project) string {
	user, _ := userFrom(r.Context())
	role, _, _ := h.svc.ProjectRole(r.Context(), user, p.ID)
	return role
}

// requireAdmin rejects non-instance-admins (used for the user directory).
func (h *Handlers) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := userFrom(r.Context())
		if !ok || !u.IsAdmin {
			WriteProblem(w, r, http.StatusForbidden, "forbidden", "Admin access required", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}
