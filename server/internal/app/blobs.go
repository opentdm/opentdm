package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/codec"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// SetBlob validates, encrypts, and replaces a file config's content at a layer,
// appending a version — all in one transaction. maxBytes (>0) caps plaintext.
func (s *Service) SetBlob(ctx context.Context, project model.Project, config model.Config, envSlug string, content []byte, maxBytes int64, comment *string, actor *uuid.UUID) (model.ConfigVersion, error) {
	if config.Kind != model.KindFile {
		return model.ConfigVersion{}, invalid("kind", "not a file config")
	}
	if config.ArchivedAt != nil {
		return model.ConfigVersion{}, ErrConflict
	}
	if maxBytes > 0 && int64(len(content)) > maxBytes {
		return model.ConfigVersion{}, invalid("content", "file exceeds size limit")
	}
	if err := codec.ValidateFile(config.Format, content); err != nil {
		return model.ConfigVersion{}, &ValidationError{Field: "content", Msg: err.Error()}
	}
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	cipher, dek, err := s.cipherAndDEK(project)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	defer zero(dek)

	hmac, err := crypto.ContentHMAC(dek, content)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	blobCT, err := cipher.Seal(content, crypto.BlobAAD(project.ID.String(), envAAD, config.ID.String()))
	if err != nil {
		return model.ConfigVersion{}, err
	}
	snapCT, err := cipher.Seal(content, crypto.VersionAAD(project.ID.String(), envAAD, config.ID.String(), model.KindFile))
	if err != nil {
		return model.ConfigVersion{}, err
	}

	var version model.ConfigVersion
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		if err := q.ReplaceBlob(ctx, config.ID, envID, store.BlobInput{
			ContentCiphertext: blobCT, DEKVersion: project.DEKVersion, ContentHMAC: hmac, SizeBytes: int64(len(content)),
		}, actor); err != nil {
			return err
		}
		version, err = s.appendVersionDedup(ctx, q, config.ID, envID, hmac, store.VersionInput{
			SnapshotKind: model.KindFile, SnapshotCiphertext: snapCT, DEKVersion: project.DEKVersion,
			ContentHMAC: hmac, ByteSize: int64(len(content)), Comment: comment, CreatedBy: actor,
		})
		return err
	})
	if err != nil {
		if isUniqueViolation(err) {
			return model.ConfigVersion{}, ErrConflict
		}
		return model.ConfigVersion{}, err
	}
	return version, nil
}

// GetBlob returns the decrypted file content at a layer, falling back from the
// requested environment to the default (base) variant when no env-specific blob
// exists.
func (s *Service) GetBlob(ctx context.Context, project model.Project, config model.Config, envSlug string) ([]byte, error) {
	if config.Kind != model.KindFile {
		return nil, invalid("kind", "not a file config")
	}
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return nil, err
	}
	blob, err := s.store.Q().GetBlob(ctx, config.ID, envID)
	usedEnvAAD := envAAD
	if err == store.ErrNotFound && envID != nil {
		blob, err = s.store.Q().GetBlob(ctx, config.ID, nil) // fall back to default variant
		usedEnvAAD = ""
	}
	if err != nil {
		return nil, err
	}
	cipher, err := s.cipherFor(project)
	if err != nil {
		return nil, err
	}
	return cipher.Open(blob.ContentCiphertext, crypto.BlobAAD(project.ID.String(), usedEnvAAD, config.ID.String()))
}
