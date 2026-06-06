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
	"github.com/opentdm/opentdm/server/internal/store"
)

// TestE2E_EnvConfigCRUD covers the rework's new backend: environment
// create/rename/set-default/reorder/delete (with guards + cascade) and config
// update/archive.
func TestE2E_EnvConfigCRUD(t *testing.T) {
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
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, out
	}
	envs := func() []environmentDTO {
		_, b := do("GET", "/projects/demo/environments", nil)
		var env struct {
			Data []environmentDTO `json:"data"`
		}
		_ = json.Unmarshal(b, &env)
		return env.Data
	}
	bySlug := func(list []environmentDTO, slug string) environmentDTO {
		for _, e := range list {
			if e.Slug == slug {
				return e
			}
		}
		return environmentDTO{}
	}

	// Bootstrap + project (seeds development/staging/production).
	if code, b := do("POST", "/auth/bootstrap", map[string]string{"setup_token": "setup-token", "username": "admin", "email": "a@b.co", "password": "supersecret"}); code != 201 {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	if code, b := do("POST", "/projects", map[string]string{"slug": "demo", "name": "Demo"}); code != 201 {
		t.Fatalf("project: %d %s", code, b)
	}
	if got := len(envs()); got != 3 {
		t.Fatalf("seed: expected 3 environments, got %d", got)
	}

	// ---- Create + rename ----
	if code, b := do("POST", "/projects/demo/environments", map[string]string{"name": "QA"}); code != 201 {
		t.Fatalf("create env: %d %s", code, b)
	}
	qa := bySlug(envs(), "qa")
	if qa.ID == "" {
		t.Fatalf("created env not found by slug 'qa'")
	}
	if code, _ := do("PATCH", "/projects/demo/environments/"+qa.ID, map[string]string{"name": "Quality", "slug": "quality"}); code != 200 {
		t.Fatalf("rename env: %d", code)
	}
	if bySlug(envs(), "quality").ID == "" {
		t.Errorf("rename did not take effect")
	}

	// ---- Duplicate slug -> 409 ----
	if code, _ := do("POST", "/projects/demo/environments", map[string]string{"slug": "staging", "name": "dup"}); code != 409 {
		t.Errorf("duplicate slug should be 409, got %d", code)
	}

	// ---- Set default ----
	staging := bySlug(envs(), "staging")
	if code, _ := do("PATCH", "/projects/demo/environments/"+staging.ID, map[string]any{"is_default": true}); code != 200 {
		t.Fatalf("set default: %d", code)
	}
	list := envs()
	if !bySlug(list, "staging").IsDefault || bySlug(list, "development").IsDefault {
		t.Errorf("default not moved to staging exactly: %+v", list)
	}

	// ---- Reorder (reverse current order) ----
	cur := envs()
	ids := make([]string, len(cur))
	for i, e := range cur {
		ids[len(cur)-1-i] = e.ID
	}
	if code, b := do("POST", "/projects/demo/environments/reorder", map[string]any{"ordered_ids": ids}); code != 200 {
		t.Fatalf("reorder: %d %s", code, b)
	}
	if got := envs()[0].ID; got != ids[0] {
		t.Errorf("reorder: first env id = %s, want %s", got, ids[0])
	}

	// ---- Cascade: items in an env are deleted when the env is deleted ----
	_, b := do("POST", "/projects/demo/configs", map[string]any{"kind": "variable", "format": "env", "name": "app"})
	var cfgEnv struct {
		Data struct{ ID string } `json:"data"`
	}
	_ = json.Unmarshal(b, &cfgEnv)
	cfgID := cfgEnv.Data.ID
	if code, _ := do("PUT", "/projects/demo/configs/"+cfgID+"/items?env=quality", map[string]any{"items": []map[string]any{{"key": "A", "value": "1"}}}); code != 200 {
		t.Fatalf("set items in quality: %d", code)
	}
	qualityID := bySlug(envs(), "quality").ID
	if code, _ := do("DELETE", "/projects/demo/environments/"+qualityID, nil); code != 200 {
		t.Fatalf("delete env: %d", code)
	}
	var itemCount int
	_ = st.Pool().QueryRow(ctx, "SELECT count(*) FROM config_items WHERE env_id = $1", qualityID).Scan(&itemCount)
	if itemCount != 0 {
		t.Errorf("cascade: expected items deleted with env, got %d", itemCount)
	}

	// ---- Keep >= 1 environment: delete down to the last one -> 422 ----
	for {
		l := envs()
		if len(l) <= 1 {
			if code, _ := do("DELETE", "/projects/demo/environments/"+l[0].ID, nil); code != 422 {
				t.Errorf("deleting the last environment should be 422, got %d", code)
			}
			break
		}
		if code, _ := do("DELETE", "/projects/demo/environments/"+l[0].ID, nil); code != 200 {
			t.Fatalf("delete env down-to-one: %d", code)
		}
	}

	// ---- Config update + archive ----
	code, _ := do("PATCH", "/projects/demo/configs/"+cfgID, map[string]any{"name": "renamed", "sort_order": 5, "tags": []string{"core", "db"}})
	if code != 200 {
		t.Fatalf("update config: %d", code)
	}
	_, b = do("GET", "/projects/demo/configs", nil)
	if !bytes.Contains(b, []byte(`"renamed"`)) || !bytes.Contains(b, []byte(`"core"`)) {
		t.Errorf("config rename/retag not reflected: %s", b)
	}
	if code, _ := do("DELETE", "/projects/demo/configs/"+cfgID, nil); code != 200 {
		t.Fatalf("archive config: %d", code)
	}
	_, b = do("GET", "/projects/demo/configs", nil)
	if bytes.Contains(b, []byte(`"renamed"`)) {
		t.Errorf("archived config should not appear in list: %s", b)
	}
}
