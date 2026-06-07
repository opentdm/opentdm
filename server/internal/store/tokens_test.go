package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TouchTokensBatch is the coalesced-flush write path. Verify the unnest-based
// batch UPDATE sets last_used_at for every id and that an empty batch is a no-op.
func TestTouchTokensBatch(t *testing.T) {
	ctx := context.Background()
	s := testDB(t)
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := s.Pool().Exec(ctx, "TRUNCATE projects RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var projectID uuid.UUID
	if err := s.Pool().QueryRow(ctx, `
		INSERT INTO projects (slug, name, dek_wrapped, dek_key_ref)
		VALUES ('p', 'P', '\x00', 'env:v1') RETURNING id`).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	mkTok := func(name string, hash []byte) uuid.UUID {
		var id uuid.UUID
		if err := s.Pool().QueryRow(ctx, `
			INSERT INTO api_tokens (project_id, name, token_prefix, token_hash)
			VALUES ($1, $2, $3, $4) RETURNING id`, projectID, name, name, hash).Scan(&id); err != nil {
			t.Fatalf("insert token %s: %v", name, err)
		}
		return id
	}
	t1 := mkTok("a", []byte("hash-a"))
	t2 := mkTok("b", []byte("hash-b"))

	at := time.Now().UTC().Truncate(time.Microsecond)
	if err := s.Q().TouchTokensBatch(ctx, []uuid.UUID{t1, t2}, []time.Time{at, at}); err != nil {
		t.Fatalf("TouchTokensBatch: %v", err)
	}
	for _, id := range []uuid.UUID{t1, t2} {
		var got *time.Time
		if err := s.Pool().QueryRow(ctx, "SELECT last_used_at FROM api_tokens WHERE id = $1", id).Scan(&got); err != nil {
			t.Fatalf("select last_used_at: %v", err)
		}
		if got == nil {
			t.Fatalf("last_used_at not set for %v", id)
		}
		if !got.Equal(at) {
			t.Fatalf("last_used_at = %v, want %v", got, at)
		}
	}

	// An empty batch must be a no-op, not an error.
	if err := s.Q().TouchTokensBatch(ctx, nil, nil); err != nil {
		t.Fatalf("empty batch should be a no-op: %v", err)
	}
}
