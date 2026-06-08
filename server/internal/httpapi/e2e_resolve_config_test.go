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

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/store"
)

// TestE2E_ResolveConfig exercises the per-file consumption primitive: resolving a
// SINGLE config (base → env override, tombstones, secrets) in isolation from the
// project's other configs, via a scoped service token.
func TestE2E_ResolveConfig(t *testing.T) {
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

	if code, b := do("POST", "/auth/bootstrap", map[string]string{
		"setup_token": "setup-token", "username": "admin", "email": "a@b.co", "password": "supersecret",
	}); code != http.StatusCreated {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	if code, b := do("POST", "/projects", map[string]string{"slug": "payments", "name": "Payments"}); code != http.StatusCreated {
		t.Fatalf("project: %d %s", code, b)
	}
	mkConfig := func(name string) string {
		code, body := do("POST", "/projects/payments/configs", map[string]any{"kind": "variable", "format": "env", "name": name})
		if code != http.StatusCreated {
			t.Fatalf("config %s: %d %s", name, code, body)
		}
		var cfg configDTO
		_ = json.Unmarshal(dataOf(body), &cfg)
		return cfg.ID
	}
	appID := mkConfig("app")
	otherID := mkConfig("other")

	// app: base PORT/LOG/DEBUG; staging overrides LOG, tombstones DEBUG, adds a secret.
	if code, b := do("PUT", "/projects/payments/configs/"+appID+"/items?env=base", map[string]any{
		"items": []map[string]any{
			{"key": "PORT", "value": "3000"},
			{"key": "LOG", "value": "info"},
			{"key": "DEBUG", "value": "true"},
		},
	}); code != http.StatusOK {
		t.Fatalf("app base: %d %s", code, b)
	}
	if code, b := do("PUT", "/projects/payments/configs/"+appID+"/items?env=staging", map[string]any{
		"items": []map[string]any{
			{"key": "LOG", "value": "debug"},
			{"key": "DEBUG", "deleted": true},
			{"key": "APIKEY", "value": "sekret", "is_secret": true},
		},
	}); code != http.StatusOK {
		t.Fatalf("app staging: %d %s", code, b)
	}
	// other: a base key that must NOT leak into app's per-file resolve.
	if code, b := do("PUT", "/projects/payments/configs/"+otherID+"/items?env=base", map[string]any{
		"items": []map[string]any{{"key": "OTHERKEY", "value": "zzz"}},
	}); code != http.StatusOK {
		t.Fatalf("other base: %d %s", code, b)
	}

	// Mint a staging-scoped read token.
	code, body := do("POST", "/projects/payments/tokens", map[string]any{
		"name": "ci-staging", "scope": "read", "environments": []string{"staging"},
	})
	if code != http.StatusCreated {
		t.Fatalf("mint token: %d %s", code, body)
	}
	var tokResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(dataOf(body), &tokResp)

	getCfg := func(configID, query string, withToken bool) (int, string) {
		req, _ := http.NewRequest("GET", base+"/projects/payments/configs/"+configID+"/resolve?"+query, nil)
		if withToken {
			req.Header.Set("Authorization", "Bearer "+tokResp.Token)
		}
		resp, err := (&http.Client{}).Do(req) // fresh client: no session cookies
		if err != nil {
			t.Fatal(err)
		}
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, string(out)
	}

	// Per-file resolve of "app" at staging via the token.
	code, out := getCfg(appID, "env=staging&format=dotenv", true)
	if code != http.StatusOK {
		t.Fatalf("resolve app/staging: %d %s", code, out)
	}
	if !strings.Contains(out, "PORT=3000") {
		t.Errorf("missing inherited base PORT=3000:\n%s", out)
	}
	if !strings.Contains(out, "LOG=debug") || strings.Contains(out, "LOG=info") {
		t.Errorf("staging override LOG should win (debug, not info):\n%s", out)
	}
	if strings.Contains(out, "DEBUG=") {
		t.Errorf("tombstone should have unset DEBUG:\n%s", out)
	}
	if !strings.Contains(out, "APIKEY=") {
		t.Errorf("secret should be included by default:\n%s", out)
	}
	if strings.Contains(out, "OTHERKEY") {
		t.Errorf("per-file resolve leaked another config's key:\n%s", out)
	}

	// include_secrets=false drops secret keys.
	if _, out := getCfg(appID, "env=staging&format=dotenv&include_secrets=false", true); strings.Contains(out, "APIKEY") {
		t.Errorf("include_secrets=false must drop APIKEY:\n%s", out)
	}

	// Out-of-scope environment is denied (default-deny).
	if code, _ := getCfg(appID, "env=production&format=dotenv", true); code != http.StatusForbidden {
		t.Errorf("out-of-scope env: got %d, want 403", code)
	}
	// Anonymous (no token, no session) is rejected uniformly.
	if code, _ := getCfg(appID, "env=staging&format=dotenv", false); code != http.StatusUnauthorized {
		t.Errorf("anonymous: got %d, want 401", code)
	}
	// Unknown config id is a 404.
	if code, _ := getCfg(uuid.NewString(), "env=staging&format=dotenv", true); code != http.StatusNotFound {
		t.Errorf("unknown config: got %d, want 404", code)
	}
}
