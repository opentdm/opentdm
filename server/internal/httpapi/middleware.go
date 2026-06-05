package httpapi

import (
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookie = "otdm_session"
	csrfCookie    = "otdm_csrf"
	csrfHeader    = "X-CSRF-Token"
)

// loadAuth populates the request context with a session user (from the cookie)
// and/or a service token (from a Bearer header) when valid. It never rejects;
// route guards decide what is required.
func (h *Handlers) loadAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
			if u, err := h.svc.AuthenticateSession(ctx, c.Value); err == nil {
				ctx = withUser(ctx, u)
			}
		}
		// Prefix-dispatch the Bearer: a user PAT (otdmu_) authenticates AS the
		// user; a service token (otdm_) is a project+env-scoped read credential.
		// The two prefixes are mutually non-prefixing, so this is unambiguous.
		if raw := bearerToken(r); raw != "" {
			switch {
			case strings.HasPrefix(raw, "otdmu_"):
				if u, err := h.svc.AuthenticatePAT(ctx, raw); err == nil {
					ctx = withUser(ctx, u)
					ctx = withPATMarker(ctx)
				}
			case strings.HasPrefix(raw, "otdm_"):
				if t, err := h.svc.AuthenticateToken(ctx, raw); err == nil {
					ctx = withToken(ctx, t)
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireSession rejects requests authenticated via a user PAT, restricting a
// route to interactive (cookie) sessions only — used for PAT lifecycle
// management so a leaked PAT cannot mint or revoke PATs.
func (h *Handlers) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPATAuth(r.Context()) {
			WriteProblem(w, r, http.StatusForbidden, "session_required", "This action requires an interactive session, not a PAT", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireUser rejects requests without a valid session user.
func (h *Handlers) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := userFrom(r.Context()); !ok {
			WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", "")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// csrf enforces a double-submit token on unsafe, cookie-authenticated requests.
// Token-authenticated (Bearer) requests carry no ambient cookie and are exempt.
func (h *Handlers) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if _, hasToken := tokenFrom(r.Context()); hasToken {
			next.ServeHTTP(w, r)
			return
		}
		if isPATAuth(r.Context()) { // PAT is a Bearer credential, no ambient cookie to forge
			next.ServeHTTP(w, r)
			return
		}
		if _, hasUser := userFrom(r.Context()); hasUser {
			c, err := r.Cookie(csrfCookie)
			if err != nil || c.Value == "" || c.Value != r.Header.Get(csrfHeader) {
				WriteProblem(w, r, http.StatusForbidden, "csrf_failed", "CSRF check failed", "")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[len("Bearer "):])
	}
	return ""
}

// setSessionCookies sets the httpOnly session cookie and the readable CSRF
// cookie (double-submit).
func (h *Handlers) setSessionCookies(w http.ResponseWriter, rawSession, csrf string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: rawSession, Path: "/",
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookie, Value: csrf, Path: "/",
		HttpOnly: false, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})
}

func (h *Handlers) clearSessionCookies(w http.ResponseWriter) {
	for _, name := range []string{sessionCookie, csrfCookie} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: name == sessionCookie})
	}
}
