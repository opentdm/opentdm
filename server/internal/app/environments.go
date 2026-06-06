package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// CreateEnvironment adds an environment (rank = max+10, not default).
func (s *Service) CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (model.Environment, error) {
	if !slugRe.MatchString(slug) {
		return model.Environment{}, invalid("slug", "must be lowercase alphanumeric/hyphen, 1-39 chars")
	}
	if name == "" {
		name = slug
	}
	var created model.Environment
	err := s.store.InTx(ctx, func(q *store.Queries) error {
		envs, err := q.ListEnvironments(ctx, projectID)
		if err != nil {
			return err
		}
		rank := 10
		for _, e := range envs {
			if e.Rank+10 > rank {
				rank = e.Rank + 10
			}
		}
		e, err := q.CreateEnvironment(ctx, model.Environment{ProjectID: projectID, Slug: slug, Name: name, Rank: rank})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		created = e
		return nil
	})
	return created, err
}

// UpdateEnvironment renames an environment (slug/name) and, when setDefault is
// true, makes it the project default. is_default cannot be unset directly — set
// another environment as default instead (keeps exactly one default).
func (s *Service) UpdateEnvironment(ctx context.Context, projectID, envID uuid.UUID, slug, name string, setDefault *bool) (model.Environment, error) {
	var updated model.Environment
	err := s.store.InTx(ctx, func(q *store.Queries) error {
		cur, err := q.GetEnvironmentByID(ctx, envID)
		if err != nil {
			return err
		}
		if cur.ProjectID != projectID {
			return ErrNotFound
		}
		if slug == "" {
			slug = cur.Slug
		}
		if name == "" {
			name = cur.Name
		}
		if !slugRe.MatchString(slug) {
			return invalid("slug", "must be lowercase alphanumeric/hyphen, 1-39 chars")
		}
		isDefault := cur.IsDefault
		if setDefault != nil && *setDefault {
			if err := q.ClearDefaultEnvironments(ctx, projectID); err != nil {
				return err
			}
			isDefault = true
		}
		e, err := q.UpdateEnvironment(ctx, projectID, model.Environment{ID: envID, Slug: slug, Name: name, Rank: cur.Rank, IsDefault: isDefault})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		updated = e
		return nil
	})
	return updated, err
}

// DeleteEnvironment removes an environment. It refuses to delete the last one
// and reassigns the default to the lowest-rank remaining environment.
func (s *Service) DeleteEnvironment(ctx context.Context, projectID, envID uuid.UUID) error {
	return s.store.InTx(ctx, func(q *store.Queries) error {
		envs, err := q.ListEnvironments(ctx, projectID) // ordered by rank, slug
		if err != nil {
			return err
		}
		var target *model.Environment
		for i := range envs {
			if envs[i].ID == envID {
				target = &envs[i]
			}
		}
		if target == nil {
			return ErrNotFound
		}
		if len(envs) <= 1 {
			return invalid("environment", "a project must keep at least one environment")
		}
		if err := q.DeleteEnvironment(ctx, projectID, envID); err != nil {
			return err
		}
		if target.IsDefault {
			for i := range envs {
				if envs[i].ID != envID {
					return q.SetDefaultEnvironment(ctx, projectID, envs[i].ID)
				}
			}
		}
		return nil
	})
}

// ReorderEnvironments rewrites ranks to match orderedIDs (10, 20, 30, …). The
// set must list every environment exactly once.
func (s *Service) ReorderEnvironments(ctx context.Context, projectID uuid.UUID, orderedIDs []uuid.UUID) ([]model.Environment, error) {
	var out []model.Environment
	err := s.store.InTx(ctx, func(q *store.Queries) error {
		envs, err := q.ListEnvironments(ctx, projectID)
		if err != nil {
			return err
		}
		if len(orderedIDs) != len(envs) {
			return invalid("ordered_ids", "must list every environment exactly once")
		}
		valid := map[uuid.UUID]bool{}
		for _, e := range envs {
			valid[e.ID] = true
		}
		seen := map[uuid.UUID]bool{}
		for i, id := range orderedIDs {
			if !valid[id] || seen[id] {
				return invalid("ordered_ids", "must list every environment exactly once")
			}
			seen[id] = true
			if err := q.SetEnvironmentRank(ctx, projectID, id, (i+1)*10); err != nil {
				return err
			}
		}
		out, err = q.ListEnvironments(ctx, projectID)
		return err
	})
	return out, err
}

// EnvTokenCount reports how many service tokens are scoped to an environment.
func (s *Service) EnvTokenCount(ctx context.Context, envID uuid.UUID) (int, error) {
	return s.store.Q().CountTokensUsingEnv(ctx, envID)
}
