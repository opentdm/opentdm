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
