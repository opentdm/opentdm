package httpapi

import "net/http"

// contentSecurityPolicy is tuned for the embedded SPA: everything loads from the
// same origin (the bundle, the API at /api, fonts/images), with 'unsafe-inline'
// for styles because @primer/react (styled-components) injects runtime <style>
// elements. No 'unsafe-eval' and no inline <script> (Vite emits a single
// same-origin bundle), so script-src stays 'self'.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self' data:; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'none'; " +
	"form-action 'self'"

// securityHeaders sets baseline response security headers on every response
// (API + the embedded SPA). HSTS is sent only when the deployment is HTTPS
// (https), since advertising it over plain HTTP is meaningless and can wedge a
// later HTTP-only setup.
func securityHeaders(https bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Content-Security-Policy", contentSecurityPolicy)
			if https {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
