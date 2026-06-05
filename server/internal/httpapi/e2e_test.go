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
	"strings"
	"testing"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/store"
)

// TestE2E_VerticalSlice exercises the whole product spine against a real
// Postgres: bootstrap -> login -> project -> config -> base+staging items ->
// mint scoped token -> resolve merged dotenv via the token.
func TestE2E_VerticalSlice(t *testing.T) {
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
	handler := NewRouter(Options{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	ts := httptest.NewServer(handler)
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	base := ts.URL + "/api/v1"
	srvURL, _ := url.Parse(ts.URL)

	csrf := func() string {
		for _, c := range jar.Cookies(srvURL) {
			if c.Name == csrfCookie {
				return c.Value
			}
		}
		return ""
	}
	do := func(method, path string, body any) (int, []byte) {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, base+path, r)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c := csrf(); c != "" {
			req.Header.Set(csrfHeader, c)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, data
	}
	dataOf := func(raw []byte) json.RawMessage {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("envelope: %v (%s)", err, raw)
		}
		return env.Data
	}

	// 1. Bootstrap the first admin (also logs in).
	if code, body := do("POST", "/auth/bootstrap", map[string]string{
		"setup_token": "setup-token", "username": "admin", "email": "a@b.co", "password": "supersecret",
	}); code != http.StatusCreated {
		t.Fatalf("bootstrap: %d %s", code, body)
	}

	// 2. Create a project.
	code, body := do("POST", "/projects", map[string]string{"slug": "payments", "name": "Payments"})
	if code != http.StatusCreated {
		t.Fatalf("create project: %d %s", code, body)
	}

	// 3. Create a variable config.
	code, body = do("POST", "/projects/payments/configs", map[string]any{
		"kind": "variable", "format": "env", "name": "app", "tags": []string{"core"},
	})
	if code != http.StatusCreated {
		t.Fatalf("create config: %d %s", code, body)
	}
	var cfg configDTO
	_ = json.Unmarshal(dataOf(body), &cfg)

	// 4. Base items + staging override.
	if code, body := do("PUT", "/projects/payments/configs/"+cfg.ID+"/items?env=base", map[string]any{
		"items": []map[string]any{
			{"key": "PORT", "value": "3000"},
			{"key": "LOG", "value": "info"},
		},
	}); code != http.StatusOK {
		t.Fatalf("set base items: %d %s", code, body)
	}
	if code, body := do("PUT", "/projects/payments/configs/"+cfg.ID+"/items?env=staging", map[string]any{
		"items": []map[string]any{{"key": "LOG", "value": "debug", "is_secret": true}},
	}); code != http.StatusOK {
		t.Fatalf("set staging items: %d %s", code, body)
	}

	// 5. Mint a staging-scoped read token.
	code, body = do("POST", "/projects/payments/tokens", map[string]any{
		"name": "ci-staging", "scope": "read", "environments": []string{"staging"},
	})
	if code != http.StatusCreated {
		t.Fatalf("mint token: %d %s", code, body)
	}
	var tokResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(dataOf(body), &tokResp)
	if !strings.HasPrefix(tokResp.Token, "otdm_") {
		t.Fatalf("unexpected token: %q", tokResp.Token)
	}

	// 6. Resolve staging via the token (no cookies) -> merged dotenv.
	tokenResolve := func(env string) (int, string) {
		req, _ := http.NewRequest("GET", base+"/projects/payments/resolve?env="+env+"&format=dotenv", nil)
		req.Header.Set("Authorization", "Bearer "+tokResp.Token)
		resp, err := (&http.Client{}).Do(req) // fresh client: token-only, no session
		if err != nil {
			t.Fatal(err)
		}
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, string(out)
	}

	code, out := tokenResolve("staging")
	if code != http.StatusOK {
		t.Fatalf("resolve staging: %d %s", code, out)
	}
	if !strings.Contains(out, "PORT=3000") {
		t.Errorf("resolve missing base value PORT=3000:\n%s", out)
	}
	if !strings.Contains(out, "LOG=debug") {
		t.Errorf("resolve should have staging override LOG=debug:\n%s", out)
	}
	if strings.Contains(out, "LOG=info") {
		t.Errorf("staging override should hide base LOG=info:\n%s", out)
	}

	// 7. The token must NOT be able to read production (out of scope).
	if code, _ := tokenResolve("production"); code != http.StatusForbidden {
		t.Errorf("expected 403 for out-of-scope env, got %d", code)
	}
}
