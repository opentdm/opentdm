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

// TestE2E_Audit covers the audit log: resource mutations are recorded with the
// right actor/action; secret values never appear in entries; a viewer can read
// the project feed but a non-member gets 404; the admin global feed spans
// projects and is admin-only; failed (403) writes are not recorded; and keyset
// pagination works.
func TestE2E_Audit(t *testing.T) {
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
	if _, err := st.Pool().Exec(ctx, "TRUNCATE users, projects, setup_singleton, audit_log RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	master, _ := crypto.RandomBytes(32)
	keys, _ := crypto.NewEnvKeyProvider("env:v1", master, nil)
	svc := app.NewService(st, keys, []byte("test-pepper"), "setup-token")
	ts := httptest.NewServer(NewRouter(Options{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	defer ts.Close()
	srvURL, _ := url.Parse(ts.URL)
	base := ts.URL + "/api/v1"

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
	mkUser := func(username, password string) *client {
		hash, _ := crypto.HashPassword(password)
		if _, err := st.Q().CreateUser(ctx, model.User{Username: username, Email: username + "@x.co", PasswordHash: hash}); err != nil {
			t.Fatalf("create %s: %v", username, err)
		}
		c := newClient()
		if code, _ := do(c, "POST", "/auth/login", map[string]string{"username": username, "password": password}); code != 200 {
			t.Fatalf("login %s", username)
		}
		return c
	}
	type auditDTO struct {
		Actor    string `json:"actor"`
		Action   string `json:"action"`
		TargetID string `json:"target_id"`
	}
	listAudit := func(c *client, path string) (int, []auditDTO, string) {
		code, b := do(c, "GET", path, nil)
		var resp struct {
			Data []auditDTO        `json:"data"`
			Meta map[string]string `json:"meta"`
		}
		_ = json.Unmarshal(b, &resp)
		return code, resp.Data, resp.Meta["next"]
	}
	hasAction := func(es []auditDTO, action string) bool {
		for _, e := range es {
			if e.Action == action {
				return true
			}
		}
		return false
	}

	// admin bootstraps + makes some mutations (incl. a secret value).
	admin := newClient()
	if code, b := do(admin, "POST", "/auth/bootstrap", map[string]string{"setup_token": "setup-token", "username": "admin", "email": "a@b.co", "password": "supersecret"}); code != 201 {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	do(admin, "POST", "/projects", map[string]string{"slug": "alpha", "name": "Alpha"})
	_, cb := do(admin, "POST", "/projects/alpha/configs", map[string]any{"kind": "variable", "format": "env", "name": "api"})
	var cfg struct {
		Data struct{ ID string } `json:"data"`
	}
	_ = json.Unmarshal(cb, &cfg)
	const secretValue = "sk_live_TOPSECRET_value"
	do(admin, "PUT", "/projects/alpha/configs/"+cfg.Data.ID+"/items?env=development", map[string]any{"items": []map[string]any{{"key": "K", "value": secretValue, "is_secret": true}}})

	// ---- project feed records actor + actions ----
	code, entries, _ := listAudit(admin, "/projects/alpha/audit")
	if code != 200 {
		t.Fatalf("admin project audit: %d", code)
	}
	if !hasAction(entries, "project.created") || !hasAction(entries, "config.created") || !hasAction(entries, "config.items.updated") {
		t.Errorf("missing expected actions: %+v", entries)
	}
	for _, e := range entries {
		if e.Actor != "admin" {
			t.Errorf("expected actor admin, got %q", e.Actor)
		}
		if e.Action == "config.created" && e.TargetID == "" {
			t.Errorf("config.created should carry a target_id")
		}
	}

	// ---- no secret value leaks into the audit feed ----
	_, raw := do(admin, "GET", "/projects/alpha/audit", nil)
	if bytes.Contains(raw, []byte(secretValue)) {
		t.Errorf("audit feed leaked a secret value")
	}

	// ---- viewer can read; non-member gets 404 ----
	bob := mkUser("bob", "bobpassword")
	bobID := func() string { u, _ := st.Q().GetUserByUsername(ctx, "bob"); return u.ID.String() }()
	do(admin, "POST", "/projects/alpha/members", map[string]any{"user": "bob", "role": "viewer"})
	if code, _, _ := listAudit(bob, "/projects/alpha/audit"); code != 200 {
		t.Errorf("viewer should read project audit, got %d", code)
	}
	carol := mkUser("carol", "carolpassword")
	if code, _ := do(carol, "GET", "/projects/alpha/audit", nil); code != 404 {
		t.Errorf("non-member project audit should be 404, got %d", code)
	}

	// ---- failed (403) write by the viewer is NOT recorded (no entry by bob) ----
	if code, _ := do(bob, "POST", "/projects/alpha/configs", map[string]any{"kind": "variable", "format": "env", "name": "nope"}); code != 403 {
		t.Fatalf("viewer write should be 403, got %d", code)
	}
	_, all, _ := listAudit(admin, "/projects/alpha/audit?limit=200")
	for _, e := range all {
		if e.Actor == "bob" {
			t.Errorf("viewer's failed write (or reads) should not be audited: %+v", e)
		}
	}
	_ = bobID

	// ---- admin global feed spans projects; non-admin is 403 ----
	if code, g, _ := listAudit(admin, "/audit"); code != 200 || !hasAction(g, "project.created") {
		t.Errorf("admin global audit: %d %+v", code, g)
	}
	if code, _ := do(bob, "GET", "/audit", nil); code != 403 {
		t.Errorf("non-admin global audit should be 403, got %d", code)
	}

	// ---- keyset pagination: limit=1 yields a cursor to the next page ----
	_, page1, next := listAudit(admin, "/projects/alpha/audit?limit=1")
	if len(page1) != 1 || next == "" {
		t.Fatalf("expected 1 entry + next cursor, got %d entries next=%q", len(page1), next)
	}
	_, page2, _ := listAudit(admin, "/projects/alpha/audit?limit=1&before="+url.QueryEscape(next))
	if len(page2) != 1 || page2[0].Action == page1[0].Action && len(all) > 1 {
		// the second page should differ from the first when more than one entry exists
		if page2[0] == page1[0] {
			t.Errorf("pagination returned the same entry twice")
		}
	}
}
