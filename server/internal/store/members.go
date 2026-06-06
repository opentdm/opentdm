package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

// AddMember upserts a user's role on a project.
func (q *Queries) AddMember(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	_, err := q.db.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, $3::project_member_role)
		ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role`,
		projectID, userID, role)
	return err
}

// GetMemberRole returns a user's role on a project, or ErrNotFound if they are
// not a member.
func (q *Queries) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (string, error) {
	var role string
	err := q.db.QueryRow(ctx,
		"SELECT role::text FROM project_members WHERE project_id = $1 AND user_id = $2",
		projectID, userID).Scan(&role)
	return role, mapNoRows(err)
}

// ListMembers returns a project's members with their usernames/emails.
func (q *Queries) ListMembers(ctx context.Context, projectID uuid.UUID) ([]model.ProjectMember, error) {
	rows, err := q.db.Query(ctx, `
		SELECT m.project_id, m.user_id, m.role::text, u.username, u.email, m.created_at
		FROM project_members m JOIN users u ON u.id = m.user_id
		WHERE m.project_id = $1
		ORDER BY m.role, u.username`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ProjectMember
	for rows.Next() {
		var m model.ProjectMember
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Role, &m.Username, &m.Email, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMemberRole sets a member's role; ErrNotFound if they are not a member.
func (q *Queries) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	ct, err := q.db.Exec(ctx,
		"UPDATE project_members SET role = $3::project_member_role WHERE project_id = $1 AND user_id = $2",
		projectID, userID, role)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RemoveMember deletes a membership; ErrNotFound if absent.
func (q *Queries) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	ct, err := q.db.Exec(ctx, "DELETE FROM project_members WHERE project_id = $1 AND user_id = $2", projectID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountOwners returns how many owners a project has (for the keep-≥1-owner guard).
func (q *Queries) CountOwners(ctx context.Context, projectID uuid.UUID) (int, error) {
	var n int
	err := q.db.QueryRow(ctx, "SELECT count(*) FROM project_members WHERE project_id = $1 AND role = 'owner'", projectID).Scan(&n)
	return n, err
}

// LockProjectMembers takes a row lock on a project's membership rows so the
// keep-≥1-owner guard and the subsequent mutation are atomic against concurrent
// demote/remove. Must run inside a transaction.
func (q *Queries) LockProjectMembers(ctx context.Context, projectID uuid.UUID) error {
	_, err := q.db.Exec(ctx, "SELECT 1 FROM project_members WHERE project_id = $1 FOR UPDATE", projectID)
	return err
}
