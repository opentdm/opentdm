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

// TestE2E_Clone covers cloning an object's content from one environment layer to
// another: variable clone with/without values (secrets + tombstones preserved),
// file clone (override-only, no base fallback), guards (from==to, unknown env,
// archived, empty source), the bulk clone-environment summary, auth, and the
// no-value-leakage property of clone responses.
func TestE2E_Clone(t *testing.T) {
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
	doRaw := func(method, path, ct string, body []byte) (int, []byte) {
		req, _ := http.NewRequest(method, base+path, bytes.NewReader(body))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
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
	createConfig := func(kind, format, name string) string {
		_, b := do("POST", "/projects/demo/configs", map[string]any{"kind": kind, "format": format, "name": name})
		var resp struct {
			Data struct{ ID string } `json:"data"`
		}
		_ = json.Unmarshal(b, &resp)
		if resp.Data.ID == "" {
			t.Fatalf("create config %s: %s", name, b)
		}
		return resp.Data.ID
	}
	putItems := func(cfgID, env string, items []map[string]any) {
		if code, b := do("PUT", "/projects/demo/configs/"+cfgID+"/items?env="+env, map[string]any{"items": items}); code != 200 {
			t.Fatalf("put items %s/%s: %d %s", cfgID, env, code, b)
		}
	}
	getItems := func(cfgID, env string) []itemDTO {
		_, b := do("GET", "/projects/demo/configs/"+cfgID+"/items?env="+env, nil)
		var resp struct {
			Data []itemDTO `json:"data"`
		}
		_ = json.Unmarshal(b, &resp)
		return resp.Data
	}
	find := func(items []itemDTO, key string) (itemDTO, bool) {
		for _, it := range items {
			if it.Key == key {
				return it, true
			}
		}
		return itemDTO{}, false
	}

	// Bootstrap + project (seeds development/staging/production).
	if code, b := do("POST", "/auth/bootstrap", map[string]string{"setup_token": "setup-token", "username": "admin", "email": "a@b.co", "password": "supersecret"}); code != 201 {
		t.Fatalf("bootstrap: %d %s", code, b)
	}
	if code, b := do("POST", "/projects", map[string]string{"slug": "demo", "name": "Demo"}); code != 201 {
		t.Fatalf("project: %d %s", code, b)
	}

	// ---- Variable config: base + development(secret + tombstone) ----
	vars := createConfig("variable", "env", "vars")
	putItems(vars, "base", []map[string]any{{"key": "COMMON", "value": "baseval"}, {"key": "KEEP", "value": "basekeep"}})
	putItems(vars, "development", []map[string]any{
		{"key": "COMMON", "value": "devval"},
		{"key": "SECRET", "value": "topsecret", "is_secret": true},
		{"key": "KEEP", "value": "", "deleted": true}, // tombstone unsetting an inherited base key
	})

	// ---- Clone dev -> staging WITH values ----
	code, body := do("POST", "/projects/demo/configs/"+vars+"/clone", map[string]any{"from": "development", "to": "staging", "with_values": true})
	if code != 200 {
		t.Fatalf("clone with values: %d %s", code, body)
	}
	if bytes.Contains(body, []byte("topsecret")) {
		t.Errorf("clone response leaked a secret value: %s", body)
	}
	st1 := getItems(vars, "staging")
	if it, ok := find(st1, "COMMON"); !ok || it.Value != "devval" {
		t.Errorf("staging COMMON = %+v, want devval", it)
	}
	if it, ok := find(st1, "SECRET"); !ok || it.Value != "topsecret" || !it.IsSecret {
		t.Errorf("staging SECRET not preserved with value+is_secret: %+v", it)
	}
	if it, ok := find(st1, "KEEP"); !ok || !it.Deleted {
		t.Errorf("staging KEEP should be a tombstone: %+v", it)
	}

	// ---- Clone dev -> production WITHOUT values ----
	if code, b := do("POST", "/projects/demo/configs/"+vars+"/clone", map[string]any{"from": "development", "to": "production", "with_values": false}); code != 200 {
		t.Fatalf("clone without values: %d %s", code, b)
	}
	pr := getItems(vars, "production")
	if it, ok := find(pr, "COMMON"); !ok || it.Value != "" {
		t.Errorf("production COMMON should be blank placeholder: %+v", it)
	}
	if it, ok := find(pr, "SECRET"); !ok || it.Value != "" || !it.IsSecret {
		t.Errorf("production SECRET should be blank but still secret: %+v", it)
	}
	if it, ok := find(pr, "KEEP"); !ok || !it.Deleted {
		t.Errorf("production KEEP should remain a tombstone: %+v", it)
	}

	// ---- Guards: from==to, unknown env ----
	if code, _ := do("POST", "/projects/demo/configs/"+vars+"/clone", map[string]any{"from": "development", "to": "development"}); code != 422 {
		t.Errorf("from==to should be 422, got %d", code)
	}
	if code, _ := do("POST", "/projects/demo/configs/"+vars+"/clone", map[string]any{"from": "nope", "to": "staging"}); code != 422 {
		t.Errorf("unknown source env should be 422, got %d", code)
	}

	// ---- Dedup: cloning identical content again cuts no new version ----
	_, b1 := do("POST", "/projects/demo/configs/"+vars+"/clone", map[string]any{"from": "development", "to": "staging", "with_values": true})
	_, b2 := do("POST", "/projects/demo/configs/"+vars+"/clone", map[string]any{"from": "development", "to": "staging", "with_values": true})
	if !bytes.Equal(extractVersion(b1), extractVersion(b2)) {
		t.Errorf("re-cloning identical content should not bump version: %s vs %s", b1, b2)
	}

	// ---- File config: base + dev override; clone override-only (no base fallback) ----
	jcfg := createConfig("file", "json", "conf")
	if code, b := doRaw("PUT", "/projects/demo/configs/"+jcfg+"/blob?env=base", "application/json", []byte(`{"base":true}`)); code != 200 {
		t.Fatalf("put base blob: %d %s", code, b)
	}
	if code, b := doRaw("PUT", "/projects/demo/configs/"+jcfg+"/blob?env=development", "application/json", []byte(`{"a":1}`)); code != 200 {
		t.Fatalf("put dev blob: %d %s", code, b)
	}
	if code, b := do("POST", "/projects/demo/configs/"+jcfg+"/clone", map[string]any{"from": "development", "to": "staging"}); code != 200 {
		t.Fatalf("clone file dev->staging: %d %s", code, b)
	}
	if _, b := doRaw("GET", "/projects/demo/configs/"+jcfg+"/blob?env=staging", "", nil); string(b) != `{"a":1}` {
		t.Errorf("staging blob = %s, want dev content", b)
	}
	// Production has no override blob (only base exists): clone must 422, proving
	// the source read is override-only and does NOT fall back to base.
	if code, _ := do("POST", "/projects/demo/configs/"+jcfg+"/clone", map[string]any{"from": "production", "to": "staging"}); code != 422 {
		t.Errorf("file clone from env without an override blob should be 422 (no base fallback), got %d", code)
	}

	// ---- Archived config -> 409 ----
	if code, _ := do("DELETE", "/projects/demo/configs/"+jcfg, nil); code != 200 {
		t.Fatalf("archive config")
	}
	if code, _ := do("POST", "/projects/demo/configs/"+jcfg+"/clone", map[string]any{"from": "development", "to": "staging"}); code != 409 {
		t.Errorf("clone into archived config should be 409, got %d", code)
	}

	// ---- Bulk clone-environment dev -> qa ----
	if code, b := do("POST", "/projects/demo/environments", map[string]string{"name": "QA"}); code != 201 {
		t.Fatalf("create qa env: %d %s", code, b)
	}
	code, b := do("POST", "/projects/demo/clone-environment", map[string]any{"from": "development", "to": "qa", "with_values": true})
	if code != 200 {
		t.Fatalf("clone-environment: %d %s", code, b)
	}
	var sum struct {
		Data struct {
			Cloned    []string `json:"cloned"`
			Unchanged []string `json:"unchanged"`
			Skipped   []string `json:"skipped"`
			Failed    []struct {
				Config string `json:"config"`
				Reason string `json:"reason"`
			} `json:"failed"`
		} `json:"data"`
	}
	_ = json.Unmarshal(b, &sum)
	if !contains(sum.Data.Cloned, "vars") {
		t.Errorf("bulk should have cloned 'vars', summary: %s", b)
	}
	if len(sum.Data.Failed) != 0 {
		t.Errorf("bulk should have no failures, got: %s", b)
	}
	if qa, ok := find(getItems(vars, "qa"), "SECRET"); !ok || qa.Value != "topsecret" || !qa.IsSecret {
		t.Errorf("qa SECRET not cloned with value+is_secret: %+v", qa)
	}

	// ---- Auth: 403 without CSRF, 401 without session ----
	req, _ := http.NewRequest("POST", base+"/projects/demo/configs/"+vars+"/clone", bytes.NewReader([]byte(`{"from":"development","to":"staging"}`)))
	req.Header.Set("Content-Type", "application/json") // logged-in jar, but no CSRF header
	if resp, _ := client.Do(req); resp.StatusCode != 403 {
		t.Errorf("clone without CSRF should be 403, got %d", resp.StatusCode)
	}
	anon := &http.Client{}
	areq, _ := http.NewRequest("POST", base+"/projects/demo/configs/"+vars+"/clone", bytes.NewReader([]byte(`{"from":"development","to":"staging"}`)))
	areq.Header.Set("Content-Type", "application/json")
	if resp, _ := anon.Do(areq); resp.StatusCode != 401 {
		t.Errorf("clone without session should be 401, got %d", resp.StatusCode)
	}
}

func extractVersion(b []byte) []byte {
	var resp struct {
		Data struct {
			Version int `json:"version"`
		} `json:"data"`
	}
	_ = json.Unmarshal(b, &resp)
	return []byte{byte(resp.Data.Version)}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
