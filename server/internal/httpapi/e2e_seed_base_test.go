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

// TestE2E_NewEnvKeySeedsBase verifies that adding a new key in an environment also
// seeds it (empty) into base, so it is inherited (empty) by every other
// environment — while a tombstone for a base-absent key does NOT seed base.
func TestE2E_NewEnvKeySeedsBase(t *testing.T) {
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
	code, body := do("POST", "/projects/payments/configs", map[string]any{"kind": "variable", "format": "env", "name": "app"})
	if code != http.StatusCreated {
		t.Fatalf("config: %d %s", code, body)
	}
	var cfg configDTO
	_ = json.Unmarshal(dataOf(body), &cfg)

	// Add a brand-new key in staging (absent from base) + a tombstone for a key that
	// doesn't exist in base (must NOT be seeded).
	if code, b := do("PUT", "/projects/payments/configs/"+cfg.ID+"/items?env=staging", map[string]any{
		"items": []map[string]any{
			{"key": "NEWVAR", "value": "val"},
			{"key": "GHOST", "deleted": true},
		},
	}); code != http.StatusOK {
		t.Fatalf("staging put: %d %s", code, b)
	}

	// Base now has NEWVAR (empty) and NOT GHOST.
	_, bb := do("GET", "/projects/payments/configs/"+cfg.ID+"/items?env=base", nil)
	var baseItems []itemDTO
	_ = json.Unmarshal(dataOf(bb), &baseItems)
	byKey := map[string]itemDTO{}
	for _, it := range baseItems {
		byKey[it.Key] = it
	}
	if v, ok := byKey["NEWVAR"]; !ok || v.Value != "" || v.Deleted {
		t.Fatalf("base should have NEWVAR empty, got %+v (all: %+v)", v, baseItems)
	}
	if _, ok := byKey["GHOST"]; ok {
		t.Errorf("tombstone GHOST must not be seeded into base: %+v", baseItems)
	}

	// development (no override) inherits NEWVAR as empty; staging keeps the override.
	if _, out := do("GET", "/projects/payments/configs/"+cfg.ID+"/resolve?env=development&format=dotenv", nil); !strings.Contains(string(out), "NEWVAR=") || strings.Contains(string(out), "NEWVAR=val") {
		t.Errorf("development resolve should inherit empty NEWVAR=:\n%s", out)
	}
	if _, out := do("GET", "/projects/payments/configs/"+cfg.ID+"/resolve?env=staging&format=dotenv", nil); !strings.Contains(string(out), "NEWVAR=val") {
		t.Errorf("staging resolve should have NEWVAR=val:\n%s", out)
	}
}
