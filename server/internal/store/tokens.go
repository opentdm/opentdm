package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const tokenSelect = `
	SELECT t.id, t.project_id, t.name, t.token_prefix, t.scope::text,
	       t.expires_at, t.last_used_at, t.revoked_at, t.created_at,
	       COALESCE(array_agg(te.environment_id) FILTER (WHERE te.environment_id IS NOT NULL), '{}') AS env_ids
	FROM api_tokens t LEFT JOIN api_token_environments te ON te.token_id = t.id `

func scanToken(row scannable) (model.Token, error) {
	var t model.Token
	err := row.Scan(&t.ID, &t.ProjectID, &t.Name, &t.Prefix, &t.Scope,
		&t.ExpiresAt, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt, &t.EnvIDs)
	return t, err
}

// CreateToken inserts a token and its environment scope. Run inside a
// transaction. envIDs must be non-empty (default-deny is enforced in service).
func (q *Queries) CreateToken(ctx context.Context, t model.Token, tokenHash []byte, createdBy *uuid.UUID) (model.Token, error) {
	var id uuid.UUID
	err := q.db.QueryRow(ctx, `
		INSERT INTO api_tokens (project_id, name, token_prefix, token_hash, scope, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5::token_scope, $6, $7)
		RETURNING id`,
		t.ProjectID, t.Name, t.Prefix, tokenHash, t.Scope, t.ExpiresAt, createdBy).Scan(&id)
	if err != nil {
		return model.Token{}, err
	}
	for _, envID := range t.EnvIDs {
		if _, err := q.db.Exec(ctx, "INSERT INTO api_token_environments (token_id, environment_id) VALUES ($1, $2)", id, envID); err != nil {
			return model.Token{}, err
		}
	}
	return q.GetTokenByID(ctx, id)
}

func (q *Queries) GetTokenByID(ctx context.Context, id uuid.UUID) (model.Token, error) {
	row := q.db.QueryRow(ctx, tokenSelect+"WHERE t.id = $1 GROUP BY t.id", id)
	t, err := scanToken(row)
	return t, mapNoRows(err)
}

// GetTokenByHash looks up a token by its hashed secret (for auth).
func (q *Queries) GetTokenByHash(ctx context.Context, tokenHash []byte) (model.Token, error) {
	row := q.db.QueryRow(ctx, tokenSelect+"WHERE t.token_hash = $1 GROUP BY t.id", tokenHash)
	t, err := scanToken(row)
	return t, mapNoRows(err)
}

func (q *Queries) ListTokens(ctx context.Context, projectID uuid.UUID) ([]model.Token, error) {
	rows, err := q.db.Query(ctx, tokenSelect+"WHERE t.project_id = $1 GROUP BY t.id ORDER BY t.created_at DESC", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RevokeToken marks a token revoked (scoped to its project).
func (q *Queries) RevokeToken(ctx context.Context, projectID, tokenID uuid.UUID) error {
	tag, err := q.db.Exec(ctx, "UPDATE api_tokens SET revoked_at = now() WHERE id = $1 AND project_id = $2 AND revoked_at IS NULL", tokenID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchToken records last use. (v1: direct update; a coalescing flusher is a
// future optimization — see DECISIONS.md.)
func (q *Queries) TouchToken(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := q.db.Exec(ctx, "UPDATE api_tokens SET last_used_at = $2 WHERE id = $1", id, at)
	return err
}
