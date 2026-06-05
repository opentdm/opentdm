package httpapi

import (
	"encoding/base64"
	"net/http"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
)

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
}

func toUserDTO(u model.User) userDTO {
	return userDTO{ID: u.ID.String(), Username: u.Username, Email: u.Email, IsAdmin: u.IsAdmin}
}

// GET /api/v1/auth/setup -> whether first-run setup is needed.
func (h *Handlers) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	needs, err := h.svc.NeedsSetup(r.Context())
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]bool{"needs_setup": needs}, nil)
}

// POST /api/v1/auth/bootstrap -> create first admin (requires setup token) and log in.
func (h *Handlers) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SetupToken string `json:"setup_token"`
		Username   string `json:"username"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	if _, err := h.svc.Bootstrap(r.Context(), req.SetupToken, req.Username, req.Email, req.Password); err != nil {
		h.writeErr(w, r, err)
		return
	}
	h.startSession(w, r, req.Username, req.Password, http.StatusCreated)
}

// POST /api/v1/auth/login
func (h *Handlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	h.startSession(w, r, req.Username, req.Password, http.StatusOK)
}

// startSession logs in and sets cookies, writing the user DTO.
func (h *Handlers) startSession(w http.ResponseWriter, r *http.Request, username, password string, status int) {
	raw, user, err := h.svc.Login(r.Context(), username, password)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	csrf, err := crypto.RandomBytes(24)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	h.setSessionCookies(w, raw, base64.RawURLEncoding.EncodeToString(csrf))
	WriteJSON(w, status, toUserDTO(user), nil)
}

// POST /api/v1/auth/logout
func (h *Handlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	h.clearSessionCookies(w)
	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true}, nil)
}

// GET /api/v1/auth/me
func (h *Handlers) handleMe(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	WriteJSON(w, http.StatusOK, toUserDTO(u), nil)
}
