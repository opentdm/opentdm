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
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// TestE2E_Phase2 covers files/blobs, versioning (history/diff/rollback), user
// PATs + write authorization, and the CSRF/requireSession guards.
func TestE2E_Phase2(t *testing.T) {
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
	handler := NewRouter(Options{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), MaxBlobBytes: 1 << 20})
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
	// session request (cookie + CSRF), JSON body
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
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, out
	}
	// raw-body session request (cookie + CSRF)
	rawReq := func(method, path, ct string, body []byte) (int, []byte) {
		req, _ := http.NewRequest(method, base+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", ct)
		if c := csrf(); c != "" {
			req.Header.Set(csrfHeader, c)
		}
		resp, _ := client.Do(req)
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, out
	}
	// Bearer request, fresh client (no cookies)
	bearer := func(method, path, token string, body any) (int, []byte) {
		var r io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, base+path, r)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, _ := (&http.Client{}).Do(req)
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, out
	}
	dataID := func(raw []byte) string {
		var env struct {
			Data struct {
				ID    string `json:"id"`
				Token string `json:"token"`
			} `json:"data"`
		}
		_ = json.Unmarshal(raw, &env)
		if env.Data.Token != "" {
			return env.Data.Token
		}
		return env.Data.ID
	}

	// Bootstrap + project.
	if code, b := do("POST", "/auth/bootstrap", map[string]string{"setup_token": "setup-token", "username": "admin", "email": "a@b.co", "password": "supersecret"}); code != 201 {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	if code, b := do("POST", "/projects", map[string]string{"slug": "p2", "name": "P2"}); code != 201 {
		t.Fatalf("project: %d %s", code, b)
	}

	// ---- Versioning on a variable config ----
	_, b := do("POST", "/projects/p2/configs", map[string]any{"kind": "variable", "format": "env", "name": "app"})
	cfg := dataID(b)
	if code, _ := do("PUT", "/projects/p2/configs/"+cfg+"/items?env=base", map[string]any{"items": []map[string]any{{"key": "A", "value": "1"}}}); code != 200 {
		t.Fatalf("set items v1: %d", code)
	}
	if code, _ := do("PUT", "/projects/p2/configs/"+cfg+"/items?env=base", map[string]any{"items": []map[string]any{{"key": "A", "value": "2"}}}); code != 200 {
		t.Fatalf("set items v2: %d", code)
	}
	_, b = do("GET", "/projects/p2/configs/"+cfg+"/versions?env=base", nil)
	var versions struct {
		Data []versionMetaDTO `json:"data"`
	}
	_ = json.Unmarshal(b, &versions)
	if len(versions.Data) != 2 {
		t.Fatalf("expected 2 versions, got %d (%s)", len(versions.Data), b)
	}
	_, b = do("GET", "/projects/p2/configs/"+cfg+"/diff?env=base&from=1&to=2", nil)
	if !strings.Contains(string(b), `"status":"changed"`) {
		t.Errorf("diff should report a changed key: %s", b)
	}
	if code, _ := do("POST", "/projects/p2/configs/"+cfg+"/rollback", map[string]any{"env": "base", "to_version": 1}); code != 200 {
		t.Fatalf("rollback: %d", code)
	}
	_, b = do("GET", "/projects/p2/configs/"+cfg+"/items?env=base", nil)
	if !strings.Contains(string(b), `"value":"1"`) {
		t.Errorf("rollback should restore A=1: %s", b)
	}

	// ---- File config: round-trip + validation ----
	// Env-only mode blocks creating file configs through the product API, so seed
	// them directly via the store — the blob round-trip + validation machinery they
	// exercise remains in the codebase.
	seedFileConfig := func(name, format string) string {
		proj, err := st.Q().GetProjectBySlug(ctx, "p2")
		if err != nil {
			t.Fatalf("get project: %v", err)
		}
		c, err := st.Q().CreateConfig(ctx, model.Config{ProjectID: proj.ID, Kind: model.KindFile, Format: format, Name: name})
		if err != nil {
			t.Fatalf("seed file config %s: %v", name, err)
		}
		return c.ID.String()
	}
	fileCfg := seedFileConfig("seed", "json")
	if code, body := rawReq("PUT", "/projects/p2/configs/"+fileCfg+"/blob?env=staging", "application/json", []byte(`{"x":1}`)); code != 200 {
		t.Fatalf("put blob: %d %s", code, body)
	}
	if code, body := rawReq("GET", "/projects/p2/configs/"+fileCfg+"/blob?env=staging", "", nil); code != 200 || string(body) != `{"x":1}` {
		t.Fatalf("get blob: %d %q", code, body)
	}
	if code, _ := rawReq("PUT", "/projects/p2/configs/"+fileCfg+"/blob?env=staging", "application/json", []byte(`{bad`)); code != 422 {
		t.Errorf("invalid JSON should be 422, got %d", code)
	}
	// XXE in an xml file config must be rejected.
	xmlCfg := seedFileConfig("doc", "xml")
	xxe := `<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY x SYSTEM "file:///etc/passwd">]><foo>&x;</foo>`
	if code, _ := rawReq("PUT", "/projects/p2/configs/"+xmlCfg+"/blob?env=base", "application/xml", []byte(xxe)); code != 422 {
		t.Errorf("XXE payload should be 422, got %d", code)
	}

	// ---- User PAT + write authorization ----
	_, b = do("POST", "/pats", map[string]any{"name": "ci"})
	pat := dataID(b)
	if !strings.HasPrefix(pat, "otdmu_") {
		t.Fatalf("unexpected PAT: %q", pat)
	}
	// PAT can write items (no cookie, no CSRF).
	if code, body := bearer("PUT", "/projects/p2/configs/"+cfg+"/items?env=base", pat, map[string]any{"items": []map[string]any{{"key": "A", "value": "9"}}}); code != 200 {
		t.Fatalf("PAT write should succeed: %d %s", code, body)
	}
	_, b = bearer("GET", "/projects/p2/configs/"+cfg+"/items?env=base", pat, nil)
	if !strings.Contains(string(b), `"value":"9"`) {
		t.Errorf("PAT write not persisted: %s", b)
	}
	// A service token must NOT be able to write (read-only).
	_, b = do("POST", "/projects/p2/tokens", map[string]any{"name": "svc", "scope": "read", "environments": []string{"staging"}})
	svcTok := dataID(b)
	if code, _ := bearer("PUT", "/projects/p2/configs/"+cfg+"/items?env=base", svcTok, map[string]any{"items": []map[string]any{}}); code != 401 {
		t.Errorf("service token write must be 401, got %d", code)
	}
	// PAT cannot manage PATs (requireSession).
	if code, _ := bearer("POST", "/pats", pat, map[string]any{"name": "evil"}); code != 403 {
		t.Errorf("PAT minting PATs must be 403, got %d", code)
	}

	// ---- CSRF still enforced for cookie writes ----
	req, _ := http.NewRequest("PUT", base+"/projects/p2/configs/"+cfg+"/items?env=base", bytes.NewReader([]byte(`{"items":[]}`)))
	req.Header.Set("Content-Type", "application/json") // no X-CSRF-Token
	resp, _ := client.Do(req)
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("cookie write without CSRF token must be 403, got %d", resp.StatusCode)
	}
}
