package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/model"
)

// resolveProject loads the {project} path param (slug or UUID), writing 404 and
// returning false if not found.
func (h *Handlers) resolveProject(w http.ResponseWriter, r *http.Request) (model.Project, bool) {
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

func (h *Handlers) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.ListProjects(r.Context())
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]projectDTO, 0, len(projects))
	for _, p := range projects {
		out = append(out, toProjectDTO(p))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

func (h *Handlers) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	user, _ := userFrom(r.Context())
	p, err := h.svc.CreateProject(r.Context(), user, req.Slug, req.Name, req.Description)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusCreated, toProjectDTO(p), nil)
}

func (h *Handlers) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, toProjectDTO(p), nil)
}

func (h *Handlers) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	envs, err := h.svc.ListEnvironments(r.Context(), p.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]environmentDTO, 0, len(envs))
	for _, e := range envs {
		out = append(out, toEnvironmentDTO(e))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}
