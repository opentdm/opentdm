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

// TestE2E_ResolveCollisions verifies the headline differentiator is surfaced:
// two configs defining the same key collide, the raw path reports the count in a
// header (unchanged), and meta=true returns the canonical envelope with full
// collision detail in meta.collisions.
func TestE2E_ResolveCollisions(t *testing.T) {
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
	if code, b := do("POST", "/projects", map[string]string{"slug": "shop", "name": "Shop"}); code != http.StatusCreated {
		t.Fatalf("project: %d %s", code, b)
	}
	mkConfig := func(name string) string {
		code, body := do("POST", "/projects/shop/configs", map[string]any{"kind": "variable", "format": "env", "name": name})
		if code != http.StatusCreated {
			t.Fatalf("config %s: %d %s", name, code, body)
		}
		var cfg configDTO
		_ = json.Unmarshal(dataOf(body), &cfg)
		return cfg.ID
	}
	c1 := mkConfig("alpha")
	c2 := mkConfig("beta")

	// Both configs define SHARED at the base layer -> a cross-config collision.
	if code, b := do("PUT", "/projects/shop/configs/"+c1+"/items?env=base", map[string]any{
		"items": []map[string]any{{"key": "SHARED", "value": "from-alpha"}, {"key": "A_ONLY", "value": "1"}},
	}); code != http.StatusOK {
		t.Fatalf("alpha items: %d %s", code, b)
	}
	if code, b := do("PUT", "/projects/shop/configs/"+c2+"/items?env=base", map[string]any{
		"items": []map[string]any{{"key": "SHARED", "value": "from-beta"}, {"key": "B_ONLY", "value": "2"}},
	}); code != http.StatusOK {
		t.Fatalf("beta items: %d %s", code, b)
	}

	// Raw path: unchanged body + collision count header.
	rawReq, _ := http.NewRequest("GET", base+"/projects/shop/resolve?env=staging&format=dotenv", nil)
	rawResp, err := client.Do(rawReq)
	if err != nil {
		t.Fatal(err)
	}
	rawBody, _ := io.ReadAll(rawResp.Body)
	rawResp.Body.Close()
	if rawResp.StatusCode != http.StatusOK {
		t.Fatalf("raw resolve: %d %s", rawResp.StatusCode, rawBody)
	}
	if got := rawResp.Header.Get("X-OpenTDM-Collisions"); got != "1" {
		t.Errorf("raw collision header = %q, want 1", got)
	}
	if !strings.Contains(string(rawBody), "SHARED=") {
		t.Errorf("raw body missing SHARED:\n%s", rawBody)
	}

	// meta=true: JSON envelope with full collision detail.
	metaReq, _ := http.NewRequest("GET", base+"/projects/shop/resolve?env=staging&meta=true", nil)
	metaResp, err := client.Do(metaReq)
	if err != nil {
		t.Fatal(err)
	}
	metaBody, _ := io.ReadAll(metaResp.Body)
	metaResp.Body.Close()
	if metaResp.StatusCode != http.StatusOK {
		t.Fatalf("meta resolve: %d %s", metaResp.StatusCode, metaBody)
	}
	if ct := metaResp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("meta content-type = %q, want application/json", ct)
	}
	if got := metaResp.Header.Get("X-OpenTDM-Collisions"); got != "1" {
		t.Errorf("meta collision header = %q, want 1", got)
	}

	type col struct {
		Key           string `json:"key"`
		WinningConfig string `json:"winning_config"`
		LosingConfig  string `json:"losing_config"`
	}
	var env struct {
		Data map[string]string `json:"data"`
		Meta struct {
			Collisions []col `json:"collisions"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(metaBody, &env); err != nil {
		t.Fatalf("decode meta envelope: %v (%s)", err, metaBody)
	}
	if len(env.Meta.Collisions) != 1 {
		t.Fatalf("want 1 collision, got %d: %s", len(env.Meta.Collisions), metaBody)
	}
	cc := env.Meta.Collisions[0]
	if cc.Key != "SHARED" {
		t.Errorf("collision key = %q, want SHARED", cc.Key)
	}
	names := map[string]bool{"alpha": true, "beta": true}
	if cc.WinningConfig == cc.LosingConfig || !names[cc.WinningConfig] || !names[cc.LosingConfig] {
		t.Errorf("unexpected winner/loser: winning=%q losing=%q", cc.WinningConfig, cc.LosingConfig)
	}
	if env.Data["SHARED"] == "" {
		t.Errorf("meta data missing SHARED: %s", metaBody)
	}
	if env.Data["A_ONLY"] != "1" || env.Data["B_ONLY"] != "2" {
		t.Errorf("meta data missing unique keys: %+v", env.Data)
	}
}
