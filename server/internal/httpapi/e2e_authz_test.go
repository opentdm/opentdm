package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// TestE2E_Authz covers per-project roles: non-member 404 (existence hidden),
// viewer reads but cannot write (403), editor writes but cannot manage members
// (403), owner manages members, the keep-≥1-owner guard (422), the admin bypass,
// per-user ListProjects filtering, and /resolve membership on the session plane
// vs. a service token's independent scope.
func TestE2E_Authz(t *testing.T) {
	dburl := os.Getenv("TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("set TEST_DATABASE_URL to run e2e tests")
	}
	ctx := context.Background()
	st, err := store.New(ctx, dburl)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := st.Pool().Exec(ctx, "TRUNCATE users, projects, setup_singleton RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	master, _ := crypto.RandomBytes(32)
	keys, _ := crypto.NewEnvKeyProvider("env:v1", master, nil)
	svc := app.NewService(st, keys, []byte("test-pepper"), "setup-token")
	ts := httptest.NewServer(NewRouter(Options{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	defer ts.Close()
	srvURL, _ := url.Parse(ts.URL)
	base := ts.URL + "/api/v1"

	// A browser-like client: its own cookie jar (session) + CSRF echo.
	type client struct {
		http *http.Client
		jar  http.CookieJar
	}
	newClient := func() *client {
		jar, _ := cookiejar.New(nil)
		return &client{http: &http.Client{Jar: jar}, jar: jar}
	}
	csrf := func(c *client) string {
		for _, ck := range c.jar.Cookies(srvURL) {
			if ck.Name == csrfCookie {
				return ck.Value
			}
		}
		return ""
	}
	do := func(c *client, method, path string, body any) (int, []byte) {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, base+path, r)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if tk := csrf(c); tk != "" {
			req.Header.Set(csrfHeader, tk)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, out
	}
	// mkUser creates a user directly (no signup endpoint) and logs them in.
	mkUser := func(username, email, password string, admin bool) *client {
		hash, _ := crypto.HashPassword(password)
		if _, err := st.Q().CreateUser(ctx, model.User{Username: username, Email: email, PasswordHash: hash, IsAdmin: admin}); err != nil {
			t.Fatalf("create user %s: %v", username, err)
		}
		c := newClient()
		if code, b := do(c, "POST", "/auth/login", map[string]string{"username": username, "password": password}); code != 200 {
			t.Fatalf("login %s: %d %s", username, code, b)
		}
		return c
	}
	userID := func(username string) string {
		u, err := st.Q().GetUserByUsername(ctx, username)
		if err != nil {
			t.Fatalf("lookup %s: %v", username, err)
		}
		return u.ID.String()
	}

	// ---- admin bootstraps and owns project "alpha" ----
	admin := newClient()
	if code, b := do(admin, "POST", "/auth/bootstrap", map[string]string{"setup_token": "setup-token", "username": "admin", "email": "admin@x.co", "password": "supersecret"}); code != 201 {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	if code, b := do(admin, "POST", "/projects", map[string]string{"slug": "alpha", "name": "Alpha"}); code != 201 {
		t.Fatalf("create alpha: %d %s", code, b)
	}

	bob := mkUser("bob", "bob@x.co", "bobpassword", false)
	bobID := userID("bob")

	// ---- non-member sees 404 (existence hidden) + empty project list ----
	if code, _ := do(bob, "GET", "/projects/alpha", nil); code != 404 {
		t.Errorf("non-member GET project should be 404, got %d", code)
	}
	if _, b := do(bob, "GET", "/projects", nil); !bytes.Contains(b, []byte("[]")) {
		t.Errorf("non-member ListProjects should be empty, got %s", b)
	}

	// ---- add bob as viewer: can read, cannot write ----
	if code, b := do(admin, "POST", "/projects/alpha/members", map[string]any{"user": "bob", "role": "viewer"}); code != 201 {
		t.Fatalf("add viewer: %d %s", code, b)
	}
	if code, b := do(bob, "GET", "/projects/alpha", nil); code != 200 || !bytes.Contains(b, []byte(`"your_role":"viewer"`)) {
		t.Errorf("viewer GET project: %d %s", code, b)
	}
	if code, _ := do(bob, "POST", "/projects/alpha/configs", map[string]any{"kind": "variable", "format": "env", "name": "app"}); code != 403 {
		t.Errorf("viewer write should be 403, got %d", code)
	}

	// ---- promote bob to editor: can write, cannot manage members ----
	if code, _ := do(admin, "PATCH", "/projects/alpha/members/"+bobID, map[string]any{"role": "editor"}); code != 200 {
		t.Fatalf("promote to editor")
	}
	if code, b := do(bob, "POST", "/projects/alpha/configs", map[string]any{"kind": "variable", "format": "env", "name": "app"}); code != 201 {
		t.Errorf("editor write should be 201, got %d %s", code, b)
	}
	if code, _ := do(bob, "POST", "/projects/alpha/members", map[string]any{"user": "admin", "role": "viewer"}); code != 403 {
		t.Errorf("editor managing members should be 403, got %d", code)
	}

	// ---- promote bob to owner: can manage members ----
	carol := mkUser("carol", "carol@x.co", "carolpassword", false)
	if code, _ := do(admin, "PATCH", "/projects/alpha/members/"+bobID, map[string]any{"role": "owner"}); code != 200 {
		t.Fatalf("promote to owner")
	}
	if code, b := do(bob, "POST", "/projects/alpha/members", map[string]any{"user": "carol", "role": "viewer"}); code != 201 {
		t.Errorf("owner adding member should be 201, got %d %s", code, b)
	}

	// ---- keep ≥1 owner: a sole owner cannot remove/demote themselves ----
	if code, b := do(bob, "POST", "/projects", map[string]string{"slug": "bobproj", "name": "Bob"}); code != 201 {
		t.Fatalf("bob create project: %d %s", code, b)
	}
	if code, _ := do(bob, "DELETE", "/projects/bobproj/members/"+bobID, nil); code != 422 {
		t.Errorf("removing the last owner should be 422, got %d", code)
	}
	if code, _ := do(bob, "PATCH", "/projects/bobproj/members/"+bobID, map[string]any{"role": "viewer"}); code != 422 {
		t.Errorf("demoting the last owner should be 422, got %d", code)
	}

	// ---- admin bypass: not a member of bobproj, yet has owner access ----
	if code, b := do(admin, "GET", "/projects/bobproj", nil); code != 200 || !bytes.Contains(b, []byte(`"your_role":"owner"`)) {
		t.Errorf("admin bypass GET bobproj: %d %s", code, b)
	}

	// ---- carol (non-member of bobproj) cannot resolve it; bob (owner) can ----
	if code, _ := do(carol, "GET", "/projects/bobproj/resolve?env=development&format=dotenv", nil); code != 404 {
		t.Errorf("non-member session resolve should be 404, got %d", code)
	}
	if code, b := do(bob, "GET", "/projects/bobproj/resolve?env=development&format=dotenv", nil); code != 200 {
		t.Errorf("owner session resolve should be 200, got %d %s", code, b)
	}

	// ---- a service token resolves regardless of who holds it (own grant) ----
	_, tb := do(bob, "POST", "/projects/bobproj/tokens", map[string]any{"name": "ci", "scope": "read", "environments": []string{"development"}})
	var tok struct {
		Data struct{ Token string } `json:"data"`
	}
	_ = json.Unmarshal(tb, &tok)
	req, _ := http.NewRequest("GET", base+"/projects/bobproj/resolve?env=development&format=dotenv", nil)
	req.Header.Set("Authorization", "Bearer "+tok.Data.Token)
	resp, err := (&http.Client{}).Do(req) // fresh client: no session, token only
	if err != nil {
		t.Fatalf("token resolve: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("service-token resolve should be 200, got %d", resp.StatusCode)
	}

	// ---- admin-only user directory ----
	if code, _ := do(bob, "GET", "/users", nil); code != 403 {
		t.Errorf("non-admin GET /users should be 403, got %d", code)
	}
	if code, b := do(admin, "GET", "/users", nil); code != 200 || !bytes.Contains(b, []byte(`"bob"`)) {
		t.Errorf("admin GET /users: %d %s", code, b)
	}

	// ---- last active admin cannot self-demote / self-deactivate (no lockout) ----
	if code, _ := do(admin, "PATCH", "/users/"+userID("admin"), map[string]any{"is_admin": false}); code != 422 {
		t.Errorf("demoting the last admin should be 422, got %d", code)
	}

	// ---- deactivating a user immediately invalidates their existing session ----
	if code, _ := do(admin, "PATCH", "/users/"+bobID, map[string]any{"is_active": false}); code != 200 {
		t.Fatalf("deactivate bob")
	}
	if code, _ := do(bob, "GET", "/projects/alpha", nil); code != 401 {
		t.Errorf("deactivated user's session should be rejected (401), got %d", code)
	}
}
