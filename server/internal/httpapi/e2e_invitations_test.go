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

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// TestE2E_Invitations covers the email-invitation onboarding flow with SMTP
// unconfigured (the accept link is returned in the response): owner creates an
// invitation, the invitee fetches it, accepts (creating their account + the
// membership) and lands logged in with the granted role; the token is single-use;
// and non-owners cannot invite.
func TestE2E_Invitations(t *testing.T) {
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
	// No Mailer → Noop; accept links are returned in the create response.
	ts := httptest.NewServer(NewRouter(Options{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	defer ts.Close()
	srvURL, _ := url.Parse(ts.URL)
	base := ts.URL + "/api/v1"

	newClient := func() (*http.Client, http.CookieJar) {
		jar, _ := cookiejar.New(nil)
		return &http.Client{Jar: jar}, jar
	}
	csrf := func(jar http.CookieJar) string {
		for _, ck := range jar.Cookies(srvURL) {
			if ck.Name == csrfCookie {
				return ck.Value
			}
		}
		return ""
	}
	do := func(c *http.Client, jar http.CookieJar, method, path string, body any) (int, []byte) {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, base+path, r)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if tk := csrf(jar); tk != "" {
			req.Header.Set(csrfHeader, tk)
		}
		resp, err := c.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, out
	}

	admin, ajar := newClient()
	if code, b := do(admin, ajar, "POST", "/auth/bootstrap", map[string]string{"setup_token": "setup-token", "username": "admin", "email": "admin@x.co", "password": "supersecret"}); code != 201 {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	if code, b := do(admin, ajar, "POST", "/projects", map[string]string{"slug": "alpha", "name": "Alpha"}); code != 201 {
		t.Fatalf("create alpha: %d %s", code, b)
	}

	// ---- owner creates an invitation; with no SMTP the accept link is returned ----
	code, b := do(admin, ajar, "POST", "/projects/alpha/invitations", map[string]any{"email": "dave@x.co", "role": "editor"})
	if code != 201 {
		t.Fatalf("create invitation: %d %s", code, b)
	}
	var inv struct {
		Data struct {
			AcceptURL string `json:"accept_url"`
			EmailSent bool   `json:"email_sent"`
		} `json:"data"`
	}
	_ = json.Unmarshal(b, &inv)
	if inv.Data.EmailSent || inv.Data.AcceptURL == "" {
		t.Fatalf("expected accept_url with email_sent=false, got %s", b)
	}
	u, _ := url.Parse(inv.Data.AcceptURL)
	token := u.Query().Get("token")
	if token == "" {
		t.Fatalf("no token in accept_url %q", inv.Data.AcceptURL)
	}

	// ---- the invitee sees the invitation details ----
	if code, b := do(admin, ajar, "GET", "/invitations/"+token, nil); code != 200 || !bytes.Contains(b, []byte(`"role":"editor"`)) || !bytes.Contains(b, []byte(`"email":"dave@x.co"`)) {
		t.Fatalf("get invitation: %d %s", code, b)
	}

	// ---- accept: creates the account + membership and logs the new user in ----
	dave, djar := newClient()
	if code, b := do(dave, djar, "POST", "/invitations/"+token+"/accept", map[string]string{"username": "dave", "password": "davepassword"}); code != 201 {
		t.Fatalf("accept: %d %s", code, b)
	}
	if code, b := do(dave, djar, "GET", "/projects/alpha", nil); code != 200 || !bytes.Contains(b, []byte(`"your_role":"editor"`)) {
		t.Errorf("accepted user should be an editor of alpha: %d %s", code, b)
	}

	// ---- single-use: the token no longer works ----
	if code, _ := do(admin, ajar, "GET", "/invitations/"+token, nil); code != 404 {
		t.Errorf("used invitation should be 404, got %d", code)
	}
	d2, d2jar := newClient()
	if code, _ := do(d2, d2jar, "POST", "/invitations/"+token+"/accept", map[string]string{"username": "dave2", "password": "dave2password"}); code == 201 {
		t.Errorf("re-accepting a used token should fail")
	}

	// ---- non-owner cannot invite ----
	hash, _ := crypto.HashPassword("bobpassword")
	if _, err := st.Q().CreateUser(ctx, model.User{Username: "bob", Email: "bob@x.co", PasswordHash: hash}); err != nil {
		t.Fatalf("create bob: %v", err)
	}
	bobUser, _ := st.Q().GetUserByUsername(ctx, "bob")
	if err := st.Q().AddMember(ctx, mustProjectID(t, st, ctx, "alpha"), bobUser.ID, "viewer"); err != nil {
		t.Fatalf("add bob viewer: %v", err)
	}
	bob, bjar := newClient()
	if code, _ := do(bob, bjar, "POST", "/auth/login", map[string]string{"username": "bob", "password": "bobpassword"}); code != 200 {
		t.Fatalf("login bob")
	}
	if code, _ := do(bob, bjar, "POST", "/projects/alpha/invitations", map[string]any{"email": "x@y.co", "role": "viewer"}); code != 403 {
		t.Errorf("viewer creating invitation should be 403, got %d", code)
	}
}

func mustProjectID(t *testing.T, st *store.Store, ctx context.Context, slug string) uuid.UUID {
	p, err := st.Q().GetProjectBySlug(ctx, slug)
	if err != nil {
		t.Fatalf("project %s: %v", slug, err)
	}
	return p.ID
}
