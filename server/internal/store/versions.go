package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

// layerPred is the WHERE fragment for "this (config, layer)" with $1=config_id
// and $2=env_id (nil → base). NULL is distinct in SQL, hence the explicit form.
const layerPred = "config_id = $1 AND (env_id = $2 OR ($2::uuid IS NULL AND env_id IS NULL))"

const versionMetaCols = `id, config_id, env_id, version, snapshot_kind::text, dek_version,
	content_hmac, byte_size, is_current, comment, created_by, created_at`
const versionFullCols = `id, config_id, env_id, version, snapshot_kind::text, snapshot_ciphertext,
	dek_version, content_hmac, byte_size, is_current, comment, created_by, created_at`

func scanVersionMeta(row scannable) (model.ConfigVersion, error) {
	var v model.ConfigVersion
	err := row.Scan(&v.ID, &v.ConfigID, &v.EnvID, &v.Version, &v.SnapshotKind, &v.DEKVersion,
		&v.ContentHMAC, &v.ByteSize, &v.IsCurrent, &v.Comment, &v.CreatedBy, &v.CreatedAt)
	return v, err
}

func scanVersionFull(row scannable) (model.ConfigVersion, error) {
	var v model.ConfigVersion
	err := row.Scan(&v.ID, &v.ConfigID, &v.EnvID, &v.Version, &v.SnapshotKind, &v.SnapshotCiphertext,
		&v.DEKVersion, &v.ContentHMAC, &v.ByteSize, &v.IsCurrent, &v.Comment, &v.CreatedBy, &v.CreatedAt)
	return v, err
}

// VersionInput is an encrypted snapshot to append.
type VersionInput struct {
	SnapshotKind       string
	SnapshotCiphertext []byte
	DEKVersion         int
	ContentHMAC        []byte
	ByteSize           int64
	Comment            *string
	CreatedBy          *uuid.UUID
}

// NextVersion returns max(version)+1 for a (config, layer); 1 if none.
func (q *Queries) NextVersion(ctx context.Context, configID uuid.UUID, envID *uuid.UUID) (int, error) {
	var n int
	err := q.db.QueryRow(ctx, "SELECT COALESCE(MAX(version),0)+1 FROM config_versions WHERE "+layerPred, configID, envID).Scan(&n)
	return n, err
}

// AppendVersion clears the prior current row for the layer and inserts the new
// row as version=n is_current=true. Run inside a transaction.
func (q *Queries) AppendVersion(ctx context.Context, configID uuid.UUID, envID *uuid.UUID, n int, in VersionInput) (model.ConfigVersion, error) {
	if _, err := q.db.Exec(ctx, "UPDATE config_versions SET is_current = false WHERE "+layerPred+" AND is_current", configID, envID); err != nil {
		return model.ConfigVersion{}, err
	}
	row := q.db.QueryRow(ctx, `
		INSERT INTO config_versions
		  (config_id, env_id, version, snapshot_kind, snapshot_ciphertext, dek_version, content_hmac, byte_size, is_current, comment, created_by)
		VALUES ($1, $2, $3, $4::config_kind, $5, $6, $7, $8, true, $9, $10)
		RETURNING `+versionFullCols,
		configID, envID, n, in.SnapshotKind, in.SnapshotCiphertext, in.DEKVersion, in.ContentHMAC, in.ByteSize, in.Comment, in.CreatedBy)
	return scanVersionFull(row)
}

// ListVersions returns version metadata (no ciphertext), newest first.
func (q *Queries) ListVersions(ctx context.Context, configID uuid.UUID, envID *uuid.UUID) ([]model.ConfigVersion, error) {
	rows, err := q.db.Query(ctx, "SELECT "+versionMetaCols+" FROM config_versions WHERE "+layerPred+" ORDER BY version DESC", configID, envID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ConfigVersion
	for rows.Next() {
		v, err := scanVersionMeta(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetVersion returns one version (with ciphertext) by number.
func (q *Queries) GetVersion(ctx context.Context, configID uuid.UUID, envID *uuid.UUID, version int) (model.ConfigVersion, error) {
	row := q.db.QueryRow(ctx, "SELECT "+versionFullCols+" FROM config_versions WHERE "+layerPred+" AND version = $3", configID, envID, version)
	v, err := scanVersionFull(row)
	return v, mapNoRows(err)
}

// GetCurrentVersion returns the current version (with ciphertext) for a layer.
func (q *Queries) GetCurrentVersion(ctx context.Context, configID uuid.UUID, envID *uuid.UUID) (model.ConfigVersion, error) {
	row := q.db.QueryRow(ctx, "SELECT "+versionFullCols+" FROM config_versions WHERE "+layerPred+" AND is_current", configID, envID)
	v, err := scanVersionFull(row)
	return v, mapNoRows(err)
}
