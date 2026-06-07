package store

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// The base layer (env_id IS NULL) needs its own partial unique index because
// NULL is distinct in a multi-column UNIQUE — without it, duplicate base keys
// would slip through. Assert both paired indexes reject duplicates and that a
// base key + an env override of the same key legitimately coexist.
func TestConfigItems_PartialUniqueIndexes(t *testing.T) {
	ctx := context.Background()
	s := testDB(t)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := s.Pool().Exec(ctx, "TRUNCATE projects RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	// Minimal fixtures via raw SQL (avoids the crypto/DEK write path).
	var projectID, envID, configID uuid.UUID
	if err := s.Pool().QueryRow(ctx, `
		INSERT INTO projects (slug, name, dek_wrapped, dek_key_ref)
		VALUES ('proj', 'Proj', '\x00', 'env:v1') RETURNING id`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if err := s.Pool().QueryRow(ctx, `
		INSERT INTO environments (project_id, slug, name) VALUES ($1, 'staging', 'Staging') RETURNING id`,
		projectID).Scan(&envID); err != nil {
		t.Fatalf("insert env: %v", err)
	}
	if err := s.Pool().QueryRow(ctx, `
		INSERT INTO configs (project_id, kind, format, name) VALUES ($1, 'variable', 'env', 'app') RETURNING id`,
		projectID).Scan(&configID); err != nil {
		t.Fatalf("insert config: %v", err)
	}

	insItem := func(env any, key string) error {
		_, err := s.Pool().Exec(ctx, `
			INSERT INTO config_items (config_id, env_id, key, value_ciphertext, dek_version)
			VALUES ($1, $2, $3, '\x00', 1)`, configID, env, key)
		return err
	}

	// Base layer: a duplicate (config_id, key) must violate uq_config_items_base.
	if err := insItem(nil, "PORT"); err != nil {
		t.Fatalf("first base PORT: %v", err)
	}
	if err := insItem(nil, "PORT"); err == nil {
		t.Fatal("duplicate base PORT should violate the base partial unique index")
	} else if !strings.Contains(err.Error(), "uq_config_items_base") {
		t.Fatalf("expected uq_config_items_base violation, got: %v", err)
	}

	// A base key and an env override of the same key must coexist (paired indexes).
	if err := insItem(envID, "PORT"); err != nil {
		t.Fatalf("env override PORT alongside base should be allowed: %v", err)
	}
	// But a duplicate env override must violate uq_config_items_env.
	if err := insItem(envID, "PORT"); err == nil {
		t.Fatal("duplicate env PORT should violate the env partial unique index")
	} else if !strings.Contains(err.Error(), "uq_config_items_env") {
		t.Fatalf("expected uq_config_items_env violation, got: %v", err)
	}
}
