package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/opentdm/opentdm/server/internal/model"
)

const blobCols = `id, config_id, env_id, content_ciphertext, dek_version, content_hmac, size_bytes, updated_by, created_at, updated_at`

func scanBlob(row scannable) (model.ConfigBlob, error) {
	var b model.ConfigBlob
	err := row.Scan(&b.ID, &b.ConfigID, &b.EnvID, &b.ContentCiphertext, &b.DEKVersion,
		&b.ContentHMAC, &b.SizeBytes, &b.UpdatedBy, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

// BlobInput is an encrypted file blob to persist.
type BlobInput struct {
	ContentCiphertext []byte
	DEKVersion        int
	ContentHMAC       []byte
	SizeBytes         int64
}

// ReplaceBlob replaces the file content at a (config, layer). envID nil targets
// the default variant. Run inside a transaction.
func (q *Queries) ReplaceBlob(ctx context.Context, configID uuid.UUID, envID *uuid.UUID, in BlobInput, updatedBy *uuid.UUID) error {
	if envID == nil {
		if _, err := q.db.Exec(ctx, "DELETE FROM config_blobs WHERE config_id = $1 AND env_id IS NULL", configID); err != nil {
			return err
		}
	} else {
		if _, err := q.db.Exec(ctx, "DELETE FROM config_blobs WHERE config_id = $1 AND env_id = $2", configID, *envID); err != nil {
			return err
		}
	}
	_, err := q.db.Exec(ctx, `
		INSERT INTO config_blobs (config_id, env_id, content_ciphertext, dek_version, content_hmac, size_bytes, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		configID, envID, in.ContentCiphertext, in.DEKVersion, in.ContentHMAC, in.SizeBytes, updatedBy)
	return err
}

// GetBlob returns the file content at one (config, layer), or ErrNotFound.
func (q *Queries) GetBlob(ctx context.Context, configID uuid.UUID, envID *uuid.UUID) (model.ConfigBlob, error) {
	sql := "SELECT " + blobCols + " FROM config_blobs WHERE config_id = $1 AND env_id IS NULL"
	args := []any{configID}
	if envID != nil {
		sql = "SELECT " + blobCols + " FROM config_blobs WHERE config_id = $1 AND env_id = $2"
		args = append(args, *envID)
	}
	b, err := scanBlob(q.db.QueryRow(ctx, sql, args...))
	return b, mapNoRows(err)
}
