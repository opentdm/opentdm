package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

// UserCount returns the number of users (used to gate first-run setup).
func (q *Queries) UserCount(ctx context.Context) (int, error) {
	var n int
	err := q.db.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&n)
	return n, err
}

// ClaimBootstrap inserts the singleton bootstrap row. The second concurrent
// caller fails on the primary key, closing the first-admin race at the DB.
func (q *Queries) ClaimBootstrap(ctx context.Context) error {
	_, err := q.db.Exec(ctx, "INSERT INTO setup_singleton (id) VALUES (true)")
	return err
}

// CreateUser inserts a user and returns the stored row.
func (q *Queries) CreateUser(ctx context.Context, u model.User) (model.User, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, password_hash, is_admin, is_active, created_at, updated_at`,
		u.Username, u.Email, u.PasswordHash, u.IsAdmin)
	return scanUser(row)
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	row := q.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, is_admin, is_active, created_at, updated_at
		FROM users WHERE username = $1`, username)
	u, err := scanUser(row)
	return u, mapNoRows(err)
}

func (q *Queries) GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	row := q.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, is_admin, is_active, created_at, updated_at
		FROM users WHERE id = $1`, id)
	u, err := scanUser(row)
	return u, mapNoRows(err)
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	row := q.db.QueryRow(ctx, `
		SELECT id, username, email, password_hash, is_admin, is_active, created_at, updated_at
		FROM users WHERE email = $1`, email)
	u, err := scanUser(row)
	return u, mapNoRows(err)
}

// ListUsers returns all users, newest first (admin directory).
func (q *Queries) ListUsers(ctx context.Context) ([]model.User, error) {
	rows, err := q.db.Query(ctx, `
		SELECT id, username, email, password_hash, is_admin, is_active, created_at, updated_at
		FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountActiveAdmins returns the number of active instance admins (for the
// last-admin guard).
func (q *Queries) CountActiveAdmins(ctx context.Context) (int, error) {
	var n int
	err := q.db.QueryRow(ctx, "SELECT count(*) FROM users WHERE is_admin AND is_active").Scan(&n)
	return n, err
}

// UpdateUserFlags sets is_active and/or is_admin (nil leaves a flag unchanged).
func (q *Queries) UpdateUserFlags(ctx context.Context, id uuid.UUID, isActive, isAdmin *bool) (model.User, error) {
	row := q.db.QueryRow(ctx, `
		UPDATE users SET
			is_active = COALESCE($2, is_active),
			is_admin  = COALESCE($3, is_admin)
		WHERE id = $1
		RETURNING id, username, email, password_hash, is_admin, is_active, created_at, updated_at`,
		id, isActive, isAdmin)
	u, err := scanUser(row)
	return u, mapNoRows(err)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanUser(row scannable) (model.User, error) {
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsAdmin, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
