package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("baseline headers always set", func(t *testing.T) {
		rec := httptest.NewRecorder()
		securityHeaders(false)(next).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		h := rec.Header()
		for k, want := range map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"Referrer-Policy":        "no-referrer",
		} {
			if got := h.Get(k); got != want {
				t.Errorf("%s = %q, want %q", k, got, want)
			}
		}
		csp := h.Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") {
			t.Errorf("CSP missing expected directives: %q", csp)
		}
		if h.Get("Strict-Transport-Security") != "" {
			t.Error("HSTS must not be sent over plain HTTP")
		}
	})

	t.Run("HSTS only on HTTPS deployments", func(t *testing.T) {
		rec := httptest.NewRecorder()
		securityHeaders(true)(next).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if !strings.HasPrefix(rec.Header().Get("Strict-Transport-Security"), "max-age=") {
			t.Errorf("expected HSTS header on HTTPS, got %q", rec.Header().Get("Strict-Transport-Security"))
		}
	})
}
