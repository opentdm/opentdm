package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const configSelect = `
	SELECT c.id, c.project_id, c.kind::text, c.format::text, c.name, c.sort_order,
	       COALESCE(c.description,''), c.is_secret, c.archived_at, c.created_at, c.updated_at
	FROM configs c `

func scanConfig(row scannable) (model.Config, error) {
	var c model.Config
	err := row.Scan(&c.ID, &c.ProjectID, &c.Kind, &c.Format, &c.Name, &c.SortOrder,
		&c.Description, &c.IsSecret, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

// CreateConfig inserts a config and returns it.
func (q *Queries) CreateConfig(ctx context.Context, c model.Config) (model.Config, error) {
	row := q.db.QueryRow(ctx, `
		INSERT INTO configs (project_id, kind, format, name, sort_order, description, is_secret, created_by)
		VALUES ($1, $2::config_kind, $3::config_format, $4, $5, $6, $7, $8)
		RETURNING id`,
		c.ProjectID, c.Kind, c.Format, c.Name, c.SortOrder, c.Description, c.IsSecret, c.CreatedBy)
	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		return model.Config{}, err
	}
	return q.GetConfig(ctx, id)
}

func (q *Queries) GetConfig(ctx context.Context, id uuid.UUID) (model.Config, error) {
	row := q.db.QueryRow(ctx, configSelect+"WHERE c.id = $1", id)
	c, err := scanConfig(row)
	return c, mapNoRows(err)
}

// UpdateConfig updates a config's name/sort_order/description.
func (q *Queries) UpdateConfig(ctx context.Context, c model.Config) (model.Config, error) {
	if _, err := q.db.Exec(ctx, "UPDATE configs SET name = $2, sort_order = $3, description = $4 WHERE id = $1",
		c.ID, c.Name, c.SortOrder, c.Description); err != nil {
		return model.Config{}, err
	}
	return q.GetConfig(ctx, c.ID)
}

// ArchiveConfig soft-deletes a config (scoped to its project).
func (q *Queries) ArchiveConfig(ctx context.Context, projectID, configID uuid.UUID) error {
	tag, err := q.db.Exec(ctx, "UPDATE configs SET archived_at = now() WHERE id = $1 AND project_id = $2 AND archived_at IS NULL", configID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchConfigs finds non-archived configs whose name matches query, across the
// projects the user can see (all of them for an admin, else those they're a
// member of). The query is expected pre-escaped for ILIKE (see app layer).
func (q *Queries) SearchConfigs(ctx context.Context, userID uuid.UUID, isAdmin bool, query string, limit int) ([]model.ConfigSearchHit, error) {
	rows, err := q.db.Query(ctx, `
		SELECT c.id, c.name, p.slug, p.name
		FROM configs c JOIN projects p ON p.id = c.project_id
		WHERE c.archived_at IS NULL AND p.archived_at IS NULL
		  AND c.name ILIKE '%' || $2 || '%' ESCAPE '\'
		  AND ($3 OR EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = p.id AND m.user_id = $1))
		ORDER BY p.name, c.sort_order, c.name
		LIMIT $4`, userID, query, isAdmin, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ConfigSearchHit
	for rows.Next() {
		var h model.ConfigSearchHit
		if err := rows.Scan(&h.ConfigID, &h.ConfigName, &h.ProjectSlug, &h.ProjectName); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ListConfigs returns non-archived configs for a project, ordered by sort_order.
func (q *Queries) ListConfigs(ctx context.Context, projectID uuid.UUID) ([]model.Config, error) {
	rows, err := q.db.Query(ctx, configSelect+
		"WHERE c.project_id = $1 AND c.archived_at IS NULL ORDER BY c.sort_order, c.name", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Config
	for rows.Next() {
		c, err := scanConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
