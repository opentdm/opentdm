package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handlers) handleListProjects(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	projects, err := h.svc.ListProjectsForUser(r.Context(), user)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]projectDTO, 0, len(projects))
	for _, p := range projects {
		out = append(out, toProjectListDTO(p))
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
	if req.Slug == "" {
		req.Slug = slugify(req.Name) // slug is derived from the name; no manual entry
	}
	user, _ := userFrom(r.Context())
	p, err := h.svc.CreateProject(r.Context(), user, req.Slug, req.Name, req.Description)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if info := auditInfoFrom(r.Context()); info != nil {
		id := p.ID
		info.ProjectID = &id
		info.TargetID = p.ID.String()
	}
	WriteJSON(w, http.StatusCreated, toProjectDTOWithRole(p, roleOwner), nil)
}

func (h *Handlers) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, toProjectDTOWithRole(p, h.callerRole(r, p)), nil)
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

// slugify derives a url-safe slug from a name.
func slugify(s string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		case !prevHyphen && b.Len() > 0:
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// envIDParam parses the {environment} UUID path param.
func envIDParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "environment"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Environment not found", "")
		return uuid.UUID{}, false
	}
	return id, true
}

// POST /projects/{project}/environments
func (h *Handlers) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	var req struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}
	env, err := h.svc.CreateEnvironment(r.Context(), p.ID, req.Slug, req.Name)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if info := auditInfoFrom(r.Context()); info != nil {
		info.TargetID = env.ID.String()
	}
	WriteJSON(w, http.StatusCreated, toEnvironmentDTO(env), nil)
}

// PATCH /projects/{project}/environments/{environment}  {slug?,name?,is_default?}
func (h *Handlers) handleUpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	id, ok := envIDParam(w, r)
	if !ok {
		return
	}
	var req struct {
		Slug      string `json:"slug"`
		Name      string `json:"name"`
		IsDefault *bool  `json:"is_default"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	env, err := h.svc.UpdateEnvironment(r.Context(), p.ID, id, req.Slug, req.Name, req.IsDefault)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toEnvironmentDTO(env), nil)
}

// DELETE /projects/{project}/environments/{environment}
func (h *Handlers) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	id, ok := envIDParam(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteEnvironment(r.Context(), p.ID, id); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

// POST /projects/{project}/environments/reorder  {ordered_ids:[...]}
func (h *Handlers) handleReorderEnvironments(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	var req struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	ids := make([]uuid.UUID, 0, len(req.OrderedIDs))
	for _, s := range req.OrderedIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			h.badRequest(w, r, "ordered_ids must be UUIDs")
			return
		}
		ids = append(ids, id)
	}
	envs, err := h.svc.ReorderEnvironments(r.Context(), p.ID, ids)
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
