package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const projectCols = `id, slug, name, COALESCE(description,''), created_by,
	dek_wrapped, dek_key_ref, dek_version, crypto_version, archived_at, created_at, updated_at`

func scanProject(row scannable) (model.Project, error) {
	var p model.Project
	err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.CreatedBy,
		&p.DEKWrapped, &p.DEKKeyRef, &p.DEKVersion, &p.CryptoVersion, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

// CreateProject inserts a project (including its wrapped DEK columns).
func (q *Queries) CreateProject(ctx context.Context, p model.Project) (model.Project, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO projects (slug, name, description, created_by, dek_wrapped, dek_key_ref, dek_version, crypto_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING `+projectCols,
		p.Slug, p.Name, p.Description, p.CreatedBy, p.DEKWrapped, p.DEKKeyRef, p.DEKVersion, p.CryptoVersion)
	return scanProject(row)
}

func (q *Queries) GetProjectByID(ctx context.Context, id uuid.UUID) (model.Project, error) {
	row := q.db.QueryRow(ctx, "SELECT "+projectCols+" FROM projects WHERE id = $1 AND archived_at IS NULL", id)
	p, err := scanProject(row)
	return p, mapNoRows(err)
}

func (q *Queries) GetProjectBySlug(ctx context.Context, slug string) (model.Project, error) {
	row := q.db.QueryRow(ctx, "SELECT "+projectCols+" FROM projects WHERE slug = $1 AND archived_at IS NULL", slug)
	p, err := scanProject(row)
	return p, mapNoRows(err)
}

// ListProjects returns non-archived projects, newest first.
func (q *Queries) ListProjects(ctx context.Context) ([]model.Project, error) {
	rows, err := q.db.Query(ctx, "SELECT "+projectCols+" FROM projects WHERE archived_at IS NULL ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
