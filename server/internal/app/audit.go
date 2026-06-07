package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

const (
	auditDefaultLimit = 50
	auditMaxLimit     = 200
)

// RecordAudit appends an audit entry. Callers treat failures as best-effort
// (the audit middleware logs them rather than failing the request).
func (s *Service) RecordAudit(ctx context.Context, e model.AuditEntry) error {
	return s.store.Q().InsertAudit(ctx, e)
}

// ListProjectAudit returns a project's audit entries, newest first, keyset-paginated.
func (s *Service) ListProjectAudit(ctx context.Context, projectID uuid.UUID, limit int, beforeTS *time.Time, beforeID *uuid.UUID) ([]model.AuditEntry, error) {
	return s.store.Q().ListAudit(ctx, &projectID, clampAuditLimit(limit), beforeTS, beforeID)
}

// ListAudit returns the instance-wide audit feed (admin), keyset-paginated.
func (s *Service) ListAudit(ctx context.Context, limit int, beforeTS *time.Time, beforeID *uuid.UUID) ([]model.AuditEntry, error) {
	return s.store.Q().ListAudit(ctx, nil, clampAuditLimit(limit), beforeTS, beforeID)
}

func clampAuditLimit(limit int) int {
	if limit <= 0 {
		return auditDefaultLimit
	}
	if limit > auditMaxLimit {
		return auditMaxLimit
	}
	return limit
}
