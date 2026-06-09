package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/opentdm/opentdm/server/internal/model"
)

func TestUpdatePreferences_RoundTrip(t *testing.T) {
	ctx := context.Background()
	s := testDB(t)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := s.Pool().Exec(ctx, "TRUNCATE users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	q := s.Q()

	u, err := q.CreateUser(ctx, model.User{Username: "alice", Email: "alice@example.com", PasswordHash: "x"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if got := string(u.Preferences); got != "{}" {
		t.Errorf("default preferences = %q, want {}", got)
	}

	updated, err := q.UpdatePreferences(ctx, u.ID, []byte(`{"color_mode":"dark","favourites":["payments","billing"]}`))
	if err != nil {
		t.Fatalf("update preferences: %v", err)
	}
	var p model.UserPreferences
	if err := json.Unmarshal(updated.Preferences, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.ColorMode != "dark" || len(p.Favourites) != 2 || p.Favourites[0] != "payments" {
		t.Errorf("preferences = %+v, want dark + [payments billing]", p)
	}

	// Re-read through a different query to confirm it persisted and that the
	// shared scanUser correctly reads the new column.
	re, err := q.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if s := string(re.Preferences); s == "" || s == "{}" {
		t.Errorf("persisted preferences empty: %q", re.Preferences)
	}
}
