package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GET /projects/{project}/members  (any member)
func (h *Handlers) handleListMembers(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	members, err := h.svc.ListMembers(r.Context(), p.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]memberDTO, 0, len(members))
	for _, m := range members {
		out = append(out, toMemberDTO(m))
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

// POST /projects/{project}/members  {user, role}  (owner)
func (h *Handlers) handleAddMember(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleOwner)
	if !ok {
		return
	}
	var req struct {
		User string `json:"user"` // username or email of an existing user
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	m, err := h.svc.AddMember(r.Context(), p.ID, req.User, req.Role)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusCreated, toMemberDTO(m), nil)
}

// PATCH /projects/{project}/members/{user}  {role}  (owner)
func (h *Handlers) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleOwner)
	if !ok {
		return
	}
	uid, ok := h.memberUserID(w, r)
	if !ok {
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	if err := h.svc.UpdateMemberRole(r.Context(), p.ID, uid, req.Role); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"user_id": uid.String(), "role": req.Role}, nil)
}

// DELETE /projects/{project}/members/{user}  (owner)
func (h *Handlers) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleOwner)
	if !ok {
		return
	}
	uid, ok := h.memberUserID(w, r)
	if !ok {
		return
	}
	if err := h.svc.RemoveMember(r.Context(), p.ID, uid); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"removed": true}, nil)
}

func (h *Handlers) memberUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "user"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Member not found", "")
		return uuid.UUID{}, false
	}
	return id, true
}
