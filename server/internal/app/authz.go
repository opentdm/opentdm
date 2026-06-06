package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// ProjectWithRole pairs a project with the requesting user's effective role.
type ProjectWithRole struct {
	Project model.Project
	Role    string
}

// ProjectRole returns the user's effective role on a project and whether they
// have any access. Instance admins (IsAdmin) are implicit owners everywhere.
func (s *Service) ProjectRole(ctx context.Context, user model.User, projectID uuid.UUID) (role string, member bool, err error) {
	if user.IsAdmin {
		return model.RoleOwner, true, nil
	}
	r, err := s.store.Q().GetMemberRole(ctx, projectID, user.ID)
	if err != nil {
		if err == store.ErrNotFound {
			return "", false, nil
		}
		return "", false, err
	}
	return r, true, nil
}

// ListProjectsForUser returns the projects a user can see — all projects for an
// admin, otherwise only those they are a member of — each with the user's role.
func (s *Service) ListProjectsForUser(ctx context.Context, user model.User) ([]ProjectWithRole, error) {
	if user.IsAdmin {
		ps, err := s.store.Q().ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]ProjectWithRole, 0, len(ps))
		for _, p := range ps {
			out = append(out, ProjectWithRole{Project: p, Role: model.RoleOwner})
		}
		return out, nil
	}
	ps, roles, err := s.store.Q().ListProjectsForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	out := make([]ProjectWithRole, 0, len(ps))
	for i, p := range ps {
		out = append(out, ProjectWithRole{Project: p, Role: roles[i]})
	}
	return out, nil
}

func validRole(role string) bool {
	return model.RoleRank(role) > 0
}
