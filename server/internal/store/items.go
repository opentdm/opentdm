package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

// ResolveItem is one variable item joined with its config, for the resolve path.
type ResolveItem struct {
	ConfigID        uuid.UUID
	ConfigName      string
	SortOrder       int
	Key             string
	ValueCiphertext []byte
	DEKVersion      int
	IsSecret        bool
	Deleted         bool
	IsBase          bool // env_id IS NULL
}

// ResolveItems returns all variable items for a project at the base layer or the
// target environment, ordered by config precedence. The caller decrypts and
// merges. Excludes archived configs.
func (q *Queries) ResolveItems(ctx context.Context, projectID, envID uuid.UUID) ([]ResolveItem, error) {
	rows, err := q.db.Query(ctx, `
		SELECT c.id, c.name, c.sort_order,
		       i.key, i.value_ciphertext, i.dek_version, i.is_secret, i.deleted, (i.env_id IS NULL) AS is_base
		FROM configs c
		JOIN config_items i ON i.config_id = c.id
		WHERE c.project_id = $1 AND c.kind = 'variable' AND c.archived_at IS NULL
		  AND (i.env_id IS NULL OR i.env_id = $2)
		ORDER BY c.sort_order, c.name`, projectID, envID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ResolveItem
	for rows.Next() {
		var it ResolveItem
		if err := rows.Scan(&it.ConfigID, &it.ConfigName, &it.SortOrder, &it.Key,
			&it.ValueCiphertext, &it.DEKVersion, &it.IsSecret, &it.Deleted, &it.IsBase); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ItemInput is an encrypted item to persist.
type ItemInput struct {
	Key             string
	ValueCiphertext []byte
	DEKVersion      int
	IsSecret        bool
	Deleted         bool
}

// ReplaceItems atomically replaces all items at a single (config, layer). Run
// inside a transaction. envID nil targets the base layer.
func (q *Queries) ReplaceItems(ctx context.Context, configID uuid.UUID, envID *uuid.UUID, items []ItemInput, updatedBy *uuid.UUID) error {
	if envID == nil {
		if _, err := q.db.Exec(ctx, "DELETE FROM config_items WHERE config_id = $1 AND env_id IS NULL", configID); err != nil {
			return err
		}
	} else {
		if _, err := q.db.Exec(ctx, "DELETE FROM config_items WHERE config_id = $1 AND env_id = $2", configID, *envID); err != nil {
			return err
		}
	}
	for _, it := range items {
		if _, err := q.db.Exec(ctx, `
			INSERT INTO config_items (config_id, env_id, key, value_ciphertext, dek_version, is_secret, deleted, updated_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			configID, envID, it.Key, it.ValueCiphertext, it.DEKVersion, it.IsSecret, it.Deleted, updatedBy); err != nil {
			return err
		}
	}
	return nil
}

// ListItems returns the items at one (config, layer) for the editor UI.
func (q *Queries) ListItems(ctx context.Context, configID uuid.UUID, envID *uuid.UUID) ([]model.ConfigItem, error) {
	sql := `SELECT id, config_id, env_id, key, value_ciphertext, dek_version, is_secret, deleted
		FROM config_items WHERE config_id = $1 AND env_id IS NULL ORDER BY key`
	args := []any{configID}
	if envID != nil {
		sql = `SELECT id, config_id, env_id, key, value_ciphertext, dek_version, is_secret, deleted
			FROM config_items WHERE config_id = $1 AND env_id = $2 ORDER BY key`
		args = append(args, *envID)
	}
	rows, err := q.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ConfigItem
	for rows.Next() {
		var it model.ConfigItem
		if err := rows.Scan(&it.ID, &it.ConfigID, &it.EnvID, &it.Key, &it.ValueCiphertext, &it.DEKVersion, &it.IsSecret, &it.Deleted); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
