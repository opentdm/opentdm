package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

// InsertAudit appends one audit entry (append-only; never updated/deleted).
func (q *Queries) InsertAudit(ctx context.Context, e model.AuditEntry) error {
	_, err := q.db.Exec(ctx, `
		INSERT INTO audit_log (project_id, actor_user_id, action, target_type, target_id, status, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ProjectID, e.ActorUserID, e.Action, nullIfEmpty(e.TargetType), nullIfEmpty(e.TargetID), e.Status, nullIfEmpty(e.IP))
	return err
}

// ListAudit returns audit entries newest-first, keyset-paginated. projectID nil
// returns the instance-wide feed; a cursor (beforeTS+beforeID) fetches the next
// page. Actor username is joined for display (empty when the actor is gone).
func (q *Queries) ListAudit(ctx context.Context, projectID *uuid.UUID, limit int, beforeTS *time.Time, beforeID *uuid.UUID) ([]model.AuditEntry, error) {
	rows, err := q.db.Query(ctx, `
		SELECT a.id, a.project_id, a.actor_user_id, COALESCE(u.username, ''), a.action,
		       COALESCE(a.target_type, ''), COALESCE(a.target_id, ''), a.status, COALESCE(a.ip, ''), a.created_at
		FROM audit_log a
		LEFT JOIN users u ON u.id = a.actor_user_id
		WHERE ($1::uuid IS NULL OR a.project_id = $1)
		  AND ($2::timestamptz IS NULL OR (a.created_at, a.id) < ($2, $3))
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $4`,
		projectID, beforeTS, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.ActorUserID, &e.ActorUsername, &e.Action,
			&e.TargetType, &e.TargetID, &e.Status, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
