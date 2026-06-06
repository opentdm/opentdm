package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const envCols = `id, project_id, slug, name, rank, is_default, created_at, updated_at`

func scanEnv(row scannable) (model.Environment, error) {
	var e model.Environment
	err := row.Scan(&e.ID, &e.ProjectID, &e.Slug, &e.Name, &e.Rank, &e.IsDefault, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

// CreateEnvironment inserts an environment.
func (q *Queries) CreateEnvironment(ctx context.Context, e model.Environment) (model.Environment, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO environments (project_id, slug, name, rank, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+envCols,
		e.ProjectID, e.Slug, e.Name, e.Rank, e.IsDefault)
	return scanEnv(row)
}

func (q *Queries) ListEnvironments(ctx context.Context, projectID uuid.UUID) ([]model.Environment, error) {
	rows, err := q.db.Query(ctx, "SELECT "+envCols+" FROM environments WHERE project_id = $1 ORDER BY rank, slug", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Environment
	for rows.Next() {
		e, err := scanEnv(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (q *Queries) GetEnvironmentBySlug(ctx context.Context, projectID uuid.UUID, slug string) (model.Environment, error) {
	row := q.db.QueryRow(ctx, "SELECT "+envCols+" FROM environments WHERE project_id = $1 AND slug = $2", projectID, slug)
	e, err := scanEnv(row)
	return e, mapNoRows(err)
}

func (q *Queries) GetEnvironmentByID(ctx context.Context, id uuid.UUID) (model.Environment, error) {
	row := q.db.QueryRow(ctx, "SELECT "+envCols+" FROM environments WHERE id = $1", id)
	e, err := scanEnv(row)
	return e, mapNoRows(err)
}

// UpdateEnvironment updates slug/name/rank/is_default for one environment,
// scoped to its project.
func (q *Queries) UpdateEnvironment(ctx context.Context, projectID uuid.UUID, e model.Environment) (model.Environment, error) {
	row := q.db.QueryRow(ctx, `
		UPDATE environments SET slug = $3, name = $4, rank = $5, is_default = $6
		WHERE id = $1 AND project_id = $2
		RETURNING `+envCols,
		e.ID, projectID, e.Slug, e.Name, e.Rank, e.IsDefault)
	out, err := scanEnv(row)
	return out, mapNoRows(err)
}

// DeleteEnvironment removes an environment (CASCADE wipes its config_items,
// config_blobs, config_versions, and api_token_environments rows).
func (q *Queries) DeleteEnvironment(ctx context.Context, projectID, id uuid.UUID) error {
	tag, err := q.db.Exec(ctx, "DELETE FROM environments WHERE id = $1 AND project_id = $2", id, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetEnvironmentRank updates only the rank of one environment.
func (q *Queries) SetEnvironmentRank(ctx context.Context, projectID, id uuid.UUID, rank int) error {
	_, err := q.db.Exec(ctx, "UPDATE environments SET rank = $3 WHERE id = $1 AND project_id = $2", id, projectID, rank)
	return err
}

// ClearDefaultEnvironments unsets is_default for all environments in a project.
func (q *Queries) ClearDefaultEnvironments(ctx context.Context, projectID uuid.UUID) error {
	_, err := q.db.Exec(ctx, "UPDATE environments SET is_default = false WHERE project_id = $1 AND is_default", projectID)
	return err
}

// SetDefaultEnvironment marks one environment as the default.
func (q *Queries) SetDefaultEnvironment(ctx context.Context, projectID, id uuid.UUID) error {
	_, err := q.db.Exec(ctx, "UPDATE environments SET is_default = true WHERE id = $1 AND project_id = $2", id, projectID)
	return err
}

// CountTokensUsingEnv reports how many service tokens are scoped to an
// environment (for the delete warning).
func (q *Queries) CountTokensUsingEnv(ctx context.Context, envID uuid.UUID) (int, error) {
	var n int
	err := q.db.QueryRow(ctx, "SELECT count(*) FROM api_token_environments WHERE environment_id = $1", envID).Scan(&n)
	return n, err
}
