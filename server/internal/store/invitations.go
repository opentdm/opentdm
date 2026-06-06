package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const invitationCols = `id, project_id, email, role::text, invited_by, expires_at, accepted_at, accepted_user_id, created_at`

func scanInvitation(row scannable) (model.Invitation, error) {
	var inv model.Invitation
	err := row.Scan(&inv.ID, &inv.ProjectID, &inv.Email, &inv.Role, &inv.InvitedBy,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.AcceptedUserID, &inv.CreatedAt)
	return inv, err
}

// CreateInvitation inserts a pending invitation.
func (q *Queries) CreateInvitation(ctx context.Context, projectID uuid.UUID, email, role string, tokenHash []byte, invitedBy *uuid.UUID, expiresAt time.Time) (model.Invitation, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO project_invitations (project_id, email, role, token_hash, invited_by, expires_at)
		VALUES ($1, $2, $3::project_member_role, $4, $5, $6)
		RETURNING `+invitationCols,
		projectID, email, role, tokenHash, invitedBy, expiresAt)
	return scanInvitation(row)
}

// GetInvitationByHash looks up an invitation by its hashed token.
func (q *Queries) GetInvitationByHash(ctx context.Context, tokenHash []byte) (model.Invitation, error) {
	row := q.db.QueryRow(ctx, "SELECT "+invitationCols+" FROM project_invitations WHERE token_hash = $1", tokenHash)
	inv, err := scanInvitation(row)
	return inv, mapNoRows(err)
}

// ListPendingInvitations returns a project's un-accepted invitations, newest first.
func (q *Queries) ListPendingInvitations(ctx context.Context, projectID uuid.UUID) ([]model.Invitation, error) {
	rows, err := q.db.Query(ctx, "SELECT "+invitationCols+" FROM project_invitations WHERE project_id = $1 AND accepted_at IS NULL ORDER BY created_at DESC", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Invitation
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// MarkInvitationAccepted records acceptance; ErrNotFound if it was already
// accepted (closes the double-accept race at the DB).
func (q *Queries) MarkInvitationAccepted(ctx context.Context, id, userID uuid.UUID, at time.Time) error {
	ct, err := q.db.Exec(ctx,
		"UPDATE project_invitations SET accepted_at = $2, accepted_user_id = $3 WHERE id = $1 AND accepted_at IS NULL",
		id, at, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteInvitation revokes a pending invitation scoped to its project.
func (q *Queries) DeleteInvitation(ctx context.Context, projectID, id uuid.UUID) error {
	ct, err := q.db.Exec(ctx, "DELETE FROM project_invitations WHERE id = $1 AND project_id = $2", id, projectID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
