package app

import (
	"context"
	"regexp"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}$`)

// seedEnvironments is the default environment set created with every project.
var seedEnvironments = []model.Environment{
	{Slug: "development", Name: "Development", Rank: 10, IsDefault: true},
	{Slug: "staging", Name: "Staging", Rank: 20},
	{Slug: "production", Name: "Production", Rank: 30},
}

// CreateProject provisions a project: it generates a per-project DEK, wraps it
// with the master key, and seeds the default environments — all in one
// transaction.
func (s *Service) CreateProject(ctx context.Context, creator model.User, slug, name, description string) (model.Project, error) {
	if !slugRe.MatchString(slug) {
		return model.Project{}, invalid("slug", "must be lowercase alphanumeric/hyphen, 1-39 chars")
	}
	if name == "" {
		name = slug
	}

	dek, err := crypto.NewDEK()
	if err != nil {
		return model.Project{}, err
	}
	wrapped, keyRef, err := s.keys.Wrap(dek)
	zero(dek)
	if err != nil {
		return model.Project{}, err
	}

	creatorID := creator.ID
	var created model.Project
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		p, err := q.CreateProject(ctx, model.Project{
			Slug: slug, Name: name, Description: description, CreatedBy: &creatorID,
			DEKWrapped: wrapped, DEKKeyRef: keyRef, DEKVersion: 1, CryptoVersion: 1,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		for _, e := range seedEnvironments {
			e.ProjectID = p.ID
			if _, err := q.CreateEnvironment(ctx, e); err != nil {
				return err
			}
		}
		created = p
		return nil
	})
	if err != nil {
		return model.Project{}, err
	}
	return created, nil
}

func (s *Service) ListProjects(ctx context.Context) ([]model.Project, error) {
	return s.store.Q().ListProjects(ctx)
}

// GetProject resolves a project by slug or UUID.
func (s *Service) GetProject(ctx context.Context, ref string) (model.Project, error) {
	if id, err := uuid.Parse(ref); err == nil {
		return s.store.Q().GetProjectByID(ctx, id)
	}
	return s.store.Q().GetProjectBySlug(ctx, ref)
}

func (s *Service) ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]model.Environment, error) {
	return s.store.Q().ListEnvironments(ctx, projectID)
}

func (s *Service) GetEnvironment(ctx context.Context, projectID uuid.UUID, slug string) (model.Environment, error) {
	return s.store.Q().GetEnvironmentBySlug(ctx, projectID, slug)
}
