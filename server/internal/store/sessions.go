package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

// CreateSession stores a new session (token already hashed by the caller).
func (q *Queries) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time) (model.Session, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at, last_seen_at`,
		userID, tokenHash, expiresAt)
	var s model.Session
	err := row.Scan(&s.ID, &s.UserID, &s.TokenHash, &s.ExpiresAt, &s.RevokedAt, &s.CreatedAt, &s.LastSeenAt)
	return s, err
}

// GetUserBySessionHash returns the user for a live (non-revoked, unexpired)
// session, or ErrNotFound.
func (q *Queries) GetUserBySessionHash(ctx context.Context, tokenHash []byte) (model.User, error) {
	row := q.db.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.password_hash, u.is_admin, u.is_active, u.created_at, u.updated_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > now() AND u.is_active`, tokenHash)
	u, err := scanUser(row)
	return u, mapNoRows(err)
}

// RevokeSession revokes a single session by its token hash.
func (q *Queries) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := q.db.Exec(ctx, "UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL", tokenHash)
	return err
}
