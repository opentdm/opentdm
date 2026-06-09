package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
)

type userDTO struct {
	ID          string          `json:"id"`
	Username    string          `json:"username"`
	Email       string          `json:"email"`
	IsAdmin     bool            `json:"is_admin"`
	Preferences json.RawMessage `json:"preferences"`
}

func toUserDTO(u model.User) userDTO {
	prefs := u.Preferences
	if len(prefs) == 0 {
		prefs = json.RawMessage("{}")
	}
	return userDTO{ID: u.ID.String(), Username: u.Username, Email: u.Email, IsAdmin: u.IsAdmin, Preferences: prefs}
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

// handleUpdateProfile updates the signed-in user's email.
func (h *Handlers) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	updated, err := h.svc.UpdateProfile(r.Context(), u.ID, req.Email)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toUserDTO(updated), nil)
}

// handleChangePassword verifies the current password and sets a new one.
func (h *Handlers) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	if err := h.svc.ChangePassword(r.Context(), u.ID, req.CurrentPassword, req.NewPassword); err != nil {
		h.writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdatePreferences replaces the signed-in user's UI preferences (theme +
// favourite project slugs). Session-authenticated; the body is the full prefs.
func (h *Handlers) handleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	u, _ := userFrom(r.Context())
	var prefs model.UserPreferences
	if err := decodeJSON(w, r, &prefs); err != nil {
		return
	}
	updated, err := h.svc.UpdatePreferences(r.Context(), u.ID, prefs)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toUserDTO(updated), nil)
}
