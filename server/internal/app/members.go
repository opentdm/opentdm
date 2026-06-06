package app

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// ListMembers returns a project's members (owner-listing is gated at the handler).
func (s *Service) ListMembers(ctx context.Context, projectID uuid.UUID) ([]model.ProjectMember, error) {
	return s.store.Q().ListMembers(ctx, projectID)
}

// AddMember grants an EXISTING user (resolved by username or email) a role on a
// project. Unknown users are rejected with a validation error so the directory
// is never enumerated for non-admins. If the change would demote the last owner
// (re-adding an existing owner at a lower role), it is refused.
func (s *Service) AddMember(ctx context.Context, projectID uuid.UUID, usernameOrEmail, role string) (model.ProjectMember, error) {
	if !validRole(role) {
		return model.ProjectMember{}, invalid("role", "must be owner, editor, or viewer")
	}
	u, err := s.lookupUser(ctx, usernameOrEmail)
	if err != nil {
		if err == store.ErrNotFound {
			return model.ProjectMember{}, invalid("user", "no such user: "+usernameOrEmail)
		}
		return model.ProjectMember{}, err
	}
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		if role != model.RoleOwner {
			if err := guardLastOwnerTx(ctx, q, projectID, u.ID); err != nil {
				return err
			}
		}
		return q.AddMember(ctx, projectID, u.ID, role)
	})
	if err != nil {
		return model.ProjectMember{}, err
	}
	return model.ProjectMember{ProjectID: projectID, UserID: u.ID, Role: role, Username: u.Username, Email: u.Email}, nil
}

// UpdateMemberRole changes a member's role, refusing to demote the last owner.
func (s *Service) UpdateMemberRole(ctx context.Context, projectID, userID uuid.UUID, role string) error {
	if !validRole(role) {
		return invalid("role", "must be owner, editor, or viewer")
	}
	return s.store.InTx(ctx, func(q *store.Queries) error {
		if role != model.RoleOwner {
			if err := guardLastOwnerTx(ctx, q, projectID, userID); err != nil {
				return err
			}
		}
		return q.UpdateMemberRole(ctx, projectID, userID, role)
	})
}

// RemoveMember removes a member, refusing to remove the last owner.
func (s *Service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return s.store.InTx(ctx, func(q *store.Queries) error {
		if err := guardLastOwnerTx(ctx, q, projectID, userID); err != nil {
			return err
		}
		return q.RemoveMember(ctx, projectID, userID)
	})
}

// guardLastOwnerTx returns a validation error if removing/demoting (project,user)
// would leave the project with no owner. It first row-locks the project's members
// so the count→mutate sequence is atomic against concurrent changes. A user who
// is not currently a member is a no-op here (the caller's mutation then decides).
func guardLastOwnerTx(ctx context.Context, q *store.Queries, projectID, userID uuid.UUID) error {
	if err := q.LockProjectMembers(ctx, projectID); err != nil {
		return err
	}
	cur, err := q.GetMemberRole(ctx, projectID, userID)
	if err == store.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if cur != model.RoleOwner {
		return nil
	}
	n, err := q.CountOwners(ctx, projectID)
	if err != nil {
		return err
	}
	if n <= 1 {
		return invalid("role", "a project must keep at least one owner")
	}
	return nil
}

// lookupUser resolves a username or email to a user.
func (s *Service) lookupUser(ctx context.Context, usernameOrEmail string) (model.User, error) {
	v := strings.TrimSpace(usernameOrEmail)
	if strings.Contains(v, "@") {
		return s.store.Q().GetUserByEmail(ctx, v)
	}
	return s.store.Q().GetUserByUsername(ctx, v)
}
