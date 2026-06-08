package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/model"
)

// resolveConfig loads the {config} UUID param and verifies it belongs to project.
func (h *Handlers) resolveConfig(w http.ResponseWriter, r *http.Request, project model.Project) (model.Config, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "config"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Config not found", "")
		return model.Config{}, false
	}
	c, err := h.svc.GetConfig(r.Context(), id)
	if err != nil || c.ProjectID != project.ID {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Config not found", "")
		return model.Config{}, false
	}
	return c, true
}

func (h *Handlers) handleListConfigs(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	configs, err := h.svc.ListConfigs(r.Context(), p.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]configDTO, 0, len(configs))
	for _, c := range configs {
		out = append(out, toConfigDTO(c))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

func (h *Handlers) handleCreateConfig(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	var req struct {
		Kind        string `json:"kind"`
		Format      string `json:"format"`
		Name        string `json:"name"`
		SortOrder   int    `json:"sort_order"`
		Description string `json:"description"`
		IsSecret    bool   `json:"is_secret"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	user, _ := userFrom(r.Context())
	c, err := h.svc.CreateConfig(r.Context(), user, p.ID, model.Config{
		Kind: req.Kind, Format: req.Format, Name: req.Name, SortOrder: req.SortOrder,
		Description: req.Description, IsSecret: req.IsSecret,
	})
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if info := auditInfoFrom(r.Context()); info != nil {
		info.TargetID = c.ID.String()
	}
	WriteJSON(w, http.StatusCreated, toConfigDTO(c), nil)
}

func (h *Handlers) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	WriteJSON(w, http.StatusOK, toConfigDTO(c), nil)
}

// PATCH /projects/{project}/configs/{config}  {name?,sort_order,description}
func (h *Handlers) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	var req struct {
		Name        string `json:"name"`
		SortOrder   int    `json:"sort_order"`
		Description string `json:"description"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	name := req.Name
	if name == "" {
		name = c.Name
	}
	updated, err := h.svc.UpdateConfig(r.Context(), p.ID, c.ID, name, req.SortOrder, req.Description)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toConfigDTO(updated), nil)
}

// DELETE /projects/{project}/configs/{config}  (soft-delete)
func (h *Handlers) handleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	if err := h.svc.ArchiveConfig(r.Context(), p.ID, c.ID); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

type itemDTO struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
	Deleted  bool   `json:"deleted"`
}

// GET /configs/{config}/items?env=  -> decrypted items at a layer (viewer+).
func (h *Handlers) handleGetItems(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	env := r.URL.Query().Get("env")
	items, err := h.svc.GetItems(r.Context(), p, c, env)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]itemDTO, 0, len(items))
	for _, it := range items {
		out = append(out, itemDTO{Key: it.Key, Value: it.Value, IsSecret: it.IsSecret, Deleted: it.Deleted})
	}
	WriteJSON(w, http.StatusOK, out, map[string]string{"env": env})
}

// PUT /configs/{config}/items?env=  -> replace all items at a layer.
func (h *Handlers) handlePutItems(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	var req struct {
		Items   []itemDTO `json:"items"`
		Comment string    `json:"comment"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	inputs := make([]app.VarInput, 0, len(req.Items))
	for _, it := range req.Items {
		inputs = append(inputs, app.VarInput{Key: it.Key, Value: it.Value, IsSecret: it.IsSecret, Deleted: it.Deleted})
	}
	env := r.URL.Query().Get("env")
	version, err := h.svc.SetItems(r.Context(), p, c, env, inputs, strPtr(req.Comment), actorID(r.Context()))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"count": len(inputs), "env": env, "version": version.Version}, nil)
}
