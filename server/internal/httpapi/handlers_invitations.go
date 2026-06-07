package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GET /projects/{project}/invitations  (owner) — pending invitations.
func (h *Handlers) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleOwner)
	if !ok {
		return
	}
	invs, err := h.svc.ListInvitations(r.Context(), p.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(invs))
	for _, inv := range invs {
		out = append(out, map[string]any{
			"id": inv.ID.String(), "email": inv.Email, "role": inv.Role, "expires_at": inv.ExpiresAt,
		})
	}
	WriteJSON(w, http.StatusOK, out, nil)
}

// POST /projects/{project}/invitations  {email, role}  (owner)
func (h *Handlers) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleOwner)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	inviter, _ := userFrom(r.Context())
	inv, raw, err := h.svc.CreateInvitation(r.Context(), p, inviter, req.Email, req.Role)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if info := auditInfoFrom(r.Context()); info != nil {
		info.TargetID = inv.ID.String()
	}
	acceptURL := h.inviteBaseURL(r) + "/accept-invite?token=" + raw
	// Always log the link so operators can recover it without email configured.
	h.logger.Info("invitation_created", "project", p.Slug, "email", req.Email, "role", req.Role, "accept_url", acceptURL)

	emailSent := false
	if h.mailer.Enabled() {
		body := fmt.Sprintf("You've been invited to the %q project on opentdm as %s.\n\nAccept and set your password:\n%s\n\nThis link expires in 7 days.", p.Name, req.Role, acceptURL)
		if err := h.mailer.Send(r.Context(), req.Email, "You're invited to "+p.Name+" on opentdm", body); err != nil {
			h.logger.Error("invitation_email_failed", "err", err)
		} else {
			emailSent = true
		}
	}
	resp := map[string]any{"id": inv.ID.String(), "email": inv.Email, "role": inv.Role, "email_sent": emailSent}
	if !emailSent {
		resp["accept_url"] = acceptURL // surfaced to the inviter when email isn't configured
	}
	WriteJSON(w, http.StatusCreated, resp, nil)
}

// DELETE /projects/{project}/invitations/{invitation}  (owner)
func (h *Handlers) handleRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleOwner)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "invitation"))
	if err != nil {
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Invitation not found", "")
		return
	}
	if err := h.svc.RevokeInvitation(r.Context(), p.ID, id); err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"revoked": true}, nil)
}

// GET /invitations/{token}  (public) — details for the accept page.
func (h *Handlers) handleGetInvitation(w http.ResponseWriter, r *http.Request) {
	inv, project, err := h.svc.InvitationByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"email": inv.Email, "role": inv.Role, "project": project.Name, "project_slug": project.Slug,
	}, nil)
}

// POST /invitations/{token}/accept  {username, password}  (public) — creates the
// account + membership and logs the new user in.
func (h *Handlers) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	if _, err := h.svc.AcceptInvitation(r.Context(), token, req.Username, req.Password); err != nil {
		h.writeErr(w, r, err)
		return
	}
	// Log the new user in with the credentials they just set.
	h.startSession(w, r, req.Username, req.Password, http.StatusCreated)
}

// inviteBaseURL returns the absolute base for invite links: the configured
// BaseURL, else derived from the request.
func (h *Handlers) inviteBaseURL(r *http.Request) string {
	if h.baseURL != "" {
		return strings.TrimRight(h.baseURL, "/")
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
