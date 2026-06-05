package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GET /api/v1/pats  -> the current user's PATs.
func (h *Handlers) handleListPATs(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	pats, err := h.svc.ListPATs(r.Context(), u.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]patDTO, 0, len(pats))
	for _, p := range pats {
		out = append(out, toPATDTO(p))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

// POST /api/v1/pats  {name, expires_in_days}  -> raw token shown once.
func (h *Handlers) handleCreatePAT(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		ExpiresInDays int    `json:"expires_in_days"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}
	u, _ := userFrom(r.Context())
	raw, pat, err := h.svc.MintPAT(r.Context(), u, req.Name, expiresAt)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	resp := struct {
		Token string `json:"token"`
		patDTO
	}{Token: raw, patDTO: toPATDTO(pat)}
	WriteJSON(w, http.StatusCreated, resp, nil)
}

// DELETE /api/v1/pats/{pat}
func (h *Handlers) handleRevokePAT(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "pat"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "PAT not found", "")
		return
	}
	u, _ := userFrom(r.Context())
	if err := h.svc.RevokePAT(r.Context(), u.ID, id); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"revoked": true}, nil)
}
