package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const patCols = `id, user_id, name, token_prefix, expires_at, last_used_at, revoked_at, created_at`

func scanPAT(row scannable) (model.UserPAT, error) {
	var p model.UserPAT
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Prefix, &p.ExpiresAt, &p.LastUsedAt, &p.RevokedAt, &p.CreatedAt)
	return p, err
}

// CreateUserPAT inserts a user PAT (token already hashed by the caller).
func (q *Queries) CreateUserPAT(ctx context.Context, p model.UserPAT, tokenHash []byte) (model.UserPAT, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO user_pats (user_id, name, token_prefix, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+patCols,
		p.UserID, p.Name, p.Prefix, tokenHash, p.ExpiresAt)
	return scanPAT(row)
}

// GetUserByPATHash returns the user for a live PAT (not revoked, not expired,
// user still active) plus the PAT id (for last-used tracking), or ErrNotFound.
func (q *Queries) GetUserByPATHash(ctx context.Context, tokenHash []byte) (model.User, uuid.UUID, error) {
	row := q.db.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.password_hash, u.is_admin, u.is_active, u.created_at, u.updated_at, p.id
		FROM user_pats p JOIN users u ON u.id = p.user_id
		WHERE p.token_hash = $1 AND p.revoked_at IS NULL
		  AND (p.expires_at IS NULL OR p.expires_at > now()) AND u.is_active`, tokenHash)
	var u model.User
	var patID uuid.UUID
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &patID)
	return u, patID, mapNoRows(err)
}

func (q *Queries) ListUserPATs(ctx context.Context, userID uuid.UUID) ([]model.UserPAT, error) {
	rows, err := q.db.Query(ctx, "SELECT "+patCols+" FROM user_pats WHERE user_id = $1 ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.UserPAT
	for rows.Next() {
		p, err := scanPAT(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RevokeUserPAT revokes a PAT scoped to its owner (cross-user revoke → ErrNotFound).
func (q *Queries) RevokeUserPAT(ctx context.Context, userID, patID uuid.UUID) error {
	tag, err := q.db.Exec(ctx, "UPDATE user_pats SET revoked_at = now() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL", patID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TouchUserPAT records last use (best-effort).
func (q *Queries) TouchUserPAT(ctx context.Context, id uuid.UUID, at time.Time) error {
	_, err := q.db.Exec(ctx, "UPDATE user_pats SET last_used_at = $2 WHERE id = $1", id, at)
	return err
}
