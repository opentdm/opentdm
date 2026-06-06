package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GET /users  (admin) — the user directory.
func (h *Handlers) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]adminUserDTO, 0, len(users))
	for _, u := range users {
		out = append(out, toAdminUserDTO(u))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

// PATCH /users/{user}  {is_active?, is_admin?}  (admin)
func (h *Handlers) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "user"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "User not found", "")
		return
	}
	var req struct {
		IsActive *bool `json:"is_active"`
		IsAdmin  *bool `json:"is_admin"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	u, err := h.svc.UpdateUser(r.Context(), id, req.IsActive, req.IsAdmin)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toAdminUserDTO(u), nil)
}
