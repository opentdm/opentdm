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

const projectColsP = `p.id, p.slug, p.name, COALESCE(p.description,''), p.created_by,
	p.dek_wrapped, p.dek_key_ref, p.dek_version, p.crypto_version, p.archived_at, p.created_at, p.updated_at`

// CountsForProjects returns object/env/member counts keyed by project id, for
// the projects-grid summary. One round-trip via correlated subqueries; absent
// ids simply don't appear in the map (zero-valued on lookup).
func (q *Queries) CountsForProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.ProjectCounts, error) {
	out := make(map[uuid.UUID]model.ProjectCounts, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.db.Query(ctx, `
		SELECT p.id,
		  (SELECT COUNT(*) FROM configs c WHERE c.project_id = p.id AND c.archived_at IS NULL),
		  (SELECT COUNT(*) FROM environments e WHERE e.project_id = p.id),
		  (SELECT COUNT(*) FROM project_members m WHERE m.project_id = p.id)
		FROM projects p WHERE p.id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var c model.ProjectCounts
		if err := rows.Scan(&id, &c.Objects, &c.Envs, &c.Members); err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, rows.Err()
}

// ListProjectsForUser returns the non-archived projects a user is a member of,
// newest first, paired with the user's role on each.
func (q *Queries) ListProjectsForUser(ctx context.Context, userID uuid.UUID) ([]model.Project, []string, error) {
	rows, err := q.db.Query(ctx, "SELECT "+projectColsP+", m.role::text "+
		"FROM projects p JOIN project_members m ON m.project_id = p.id "+
		"WHERE m.user_id = $1 AND p.archived_at IS NULL ORDER BY p.created_at DESC", userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var projects []model.Project
	var roles []string
	for rows.Next() {
		var p model.Project
		var role string
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.CreatedBy,
			&p.DEKWrapped, &p.DEKKeyRef, &p.DEKVersion, &p.CryptoVersion, &p.ArchivedAt, &p.CreatedAt, &p.UpdatedAt,
			&role); err != nil {
			return nil, nil, err
		}
		projects = append(projects, p)
		roles = append(roles, role)
	}
	return projects, roles, rows.Err()
}
