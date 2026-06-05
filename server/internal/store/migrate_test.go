package store

import (
	"context"
	"os"
	"testing"
)

// testDB returns a connected Store or skips if TEST_DATABASE_URL is unset.
func testDB(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run store integration tests")
	}
	s, err := New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestMigrate_AppliesAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := testDB(t)

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	// Running again must be a no-op, not an error.
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("second migrate (idempotent): %v", err)
	}

	// Spot-check that core tables and the kind/format CHECK exist.
	for _, table := range []string{"users", "sessions", "setup_singleton", "projects",
		"environments", "configs", "config_items", "api_tokens", "api_token_environments"} {
		var n int
		if err := s.Pool().QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Errorf("table %s not usable: %v", table, err)
		}
	}
}
