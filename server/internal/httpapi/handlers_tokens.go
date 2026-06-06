package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

func (h *Handlers) handleListTokens(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	tokens, err := h.svc.ListTokens(r.Context(), p.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]tokenDTO, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, toTokenDTO(t))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

// POST /projects/{project}/tokens -> mint a token; the raw secret is shown once.
func (h *Handlers) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	var req struct {
		Name          string   `json:"name"`
		Scope         string   `json:"scope"`
		Environments  []string `json:"environments"`
		ExpiresInDays int      `json:"expires_in_days"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	if req.Scope == "" {
		req.Scope = model.ScopeRead
	}
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}
	user, _ := userFrom(r.Context())
	raw, token, err := h.svc.MintToken(r.Context(), user, p, req.Name, req.Scope, req.Environments, expiresAt)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	resp := struct {
		Token string `json:"token"`
		tokenDTO
	}{Token: raw, tokenDTO: toTokenDTO(token)}
	WriteJSON(w, http.StatusCreated, resp, nil)
}

// DELETE /projects/{project}/tokens/{token}
func (h *Handlers) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "token"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Token not found", "")
		return
	}
	if err := h.svc.RevokeToken(r.Context(), p.ID, id); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"revoked": true}, nil)
}
