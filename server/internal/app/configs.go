package app

import (
	"bytes"
	"context"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/codec"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

var variableFormats = map[string]bool{
	model.FormatEnv: true, model.FormatProperties: true, model.FormatSecret: true,
}
var fileFormats = map[string]bool{
	model.FormatJSON: true, model.FormatCSV: true, model.FormatXML: true,
}

// CreateConfig creates a config (and tags) after validating the kind/format
// pairing.
func (s *Service) CreateConfig(ctx context.Context, creator model.User, projectID uuid.UUID, c model.Config) (model.Config, error) {
	if c.Name == "" {
		return model.Config{}, invalid("name", "must not be empty")
	}
	switch c.Kind {
	case model.KindVariable:
		if !variableFormats[c.Format] {
			return model.Config{}, invalid("format", "variable configs must be env, properties, or secret")
		}
	case model.KindFile:
		if !fileFormats[c.Format] {
			return model.Config{}, invalid("format", "file configs must be json, csv, or xml")
		}
	default:
		return model.Config{}, invalid("kind", "must be variable or file")
	}

	c.ProjectID = projectID
	c.CreatedBy = &creator.ID
	var created model.Config
	err := s.store.InTx(ctx, func(q *store.Queries) error {
		out, err := q.CreateConfig(ctx, c)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		created = out
		return nil
	})
	return created, err
}

func (s *Service) ListConfigs(ctx context.Context, projectID uuid.UUID) ([]model.Config, error) {
	return s.store.Q().ListConfigs(ctx, projectID)
}

// UpdateConfig renames/retags a config and sets its sort_order/description.
func (s *Service) UpdateConfig(ctx context.Context, projectID, configID uuid.UUID, name string, sortOrder int, description string, tags []string) (model.Config, error) {
	if name == "" {
		return model.Config{}, invalid("name", "must not be empty")
	}
	var out model.Config
	err := s.store.InTx(ctx, func(q *store.Queries) error {
		cur, err := q.GetConfig(ctx, configID)
		if err != nil {
			return err
		}
		if cur.ProjectID != projectID {
			return ErrNotFound
		}
		c, err := q.UpdateConfig(ctx, model.Config{ID: configID, Name: name, SortOrder: sortOrder, Description: description, Tags: tags})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		out = c
		return nil
	})
	return out, err
}

// ArchiveConfig soft-deletes a config.
func (s *Service) ArchiveConfig(ctx context.Context, projectID, configID uuid.UUID) error {
	return s.store.Q().ArchiveConfig(ctx, projectID, configID)
}

func (s *Service) GetConfig(ctx context.Context, id uuid.UUID) (model.Config, error) {
	return s.store.Q().GetConfig(ctx, id)
}

// VarInput is a plaintext variable to persist at a layer.
type VarInput struct {
	Key      string
	Value    string
	IsSecret bool
	Deleted  bool
}

// VarOutput is a decrypted variable for the editor UI.
type VarOutput struct {
	Key      string
	Value    string
	IsSecret bool
	Deleted  bool
}

// SetItems encrypts and replaces all items at one (config, layer) AND appends a
// version snapshot — both in one transaction. envSlug "" or "base" targets the
// base layer. Returns the resulting current version.
func (s *Service) SetItems(ctx context.Context, project model.Project, config model.Config, envSlug string, inputs []VarInput, comment *string, actor *uuid.UUID) (model.ConfigVersion, error) {
	if config.ArchivedAt != nil {
		return model.ConfigVersion{}, ErrConflict
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

	items := make([]store.ItemInput, 0, len(inputs))
	snap := make([]codec.SnapshotItem, 0, len(inputs))
	for _, in := range inputs {
		if !codec.ValidKey(in.Key) {
			return model.ConfigVersion{}, invalid("key", "invalid variable name: "+in.Key)
		}
		plaintext := in.Value
		if in.Deleted {
			plaintext = "" // tombstone carries no value
		}
		ct, err := cipher.Seal([]byte(plaintext), crypto.ItemAAD(project.ID.String(), envAAD, config.ID.String(), in.Key))
		if err != nil {
			return model.ConfigVersion{}, err
		}
		items = append(items, store.ItemInput{
			Key: in.Key, ValueCiphertext: ct, DEKVersion: project.DEKVersion,
			IsSecret: in.IsSecret, Deleted: in.Deleted,
		})
		snap = append(snap, codec.SnapshotItem{Key: in.Key, Value: in.Value, IsSecret: in.IsSecret, Deleted: in.Deleted})
	}

	canon, err := codec.CanonicalVarSnapshot(snap)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	hmac, err := crypto.ContentHMAC(dek, canon)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	snapCT, err := cipher.Seal(canon, crypto.VersionAAD(project.ID.String(), envAAD, config.ID.String(), model.KindVariable))
	if err != nil {
		return model.ConfigVersion{}, err
	}

	var version model.ConfigVersion
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		if err := q.ReplaceItems(ctx, config.ID, envID, items, actor); err != nil {
			return err
		}
		version, err = s.appendVersionDedup(ctx, q, config.ID, envID, hmac, store.VersionInput{
			SnapshotKind: model.KindVariable, SnapshotCiphertext: snapCT, DEKVersion: project.DEKVersion,
			ContentHMAC: hmac, ByteSize: int64(len(canon)), Comment: comment, CreatedBy: actor,
		})
		return err
	})
	if err != nil {
		if isUniqueViolation(err) {
			return model.ConfigVersion{}, ErrConflict // concurrent save lost the version-number race
		}
		return model.ConfigVersion{}, err
	}
	return version, nil
}

// appendVersionDedup appends a new version unless the layer's current version
// already has the same content hash (a no-op save), in which case it returns the
// current version without spamming history. Runs inside the caller's tx.
func (s *Service) appendVersionDedup(ctx context.Context, q *store.Queries, configID uuid.UUID, envID *uuid.UUID, hmac []byte, in store.VersionInput) (model.ConfigVersion, error) {
	cur, err := q.GetCurrentVersion(ctx, configID, envID)
	if err == nil && bytes.Equal(cur.ContentHMAC, hmac) {
		return cur, nil
	}
	if err != nil && err != store.ErrNotFound {
		return model.ConfigVersion{}, err
	}
	n, err := q.NextVersion(ctx, configID, envID)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	return q.AppendVersion(ctx, configID, envID, n, in)
}

// GetItems returns decrypted items at one (config, layer) for the editor/viewer.
func (s *Service) GetItems(ctx context.Context, project model.Project, config model.Config, envSlug string) ([]VarOutput, error) {
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.Q().ListItems(ctx, config.ID, envID)
	if err != nil {
		return nil, err
	}
	cipher, err := s.cipherFor(project)
	if err != nil {
		return nil, err
	}
	out := make([]VarOutput, 0, len(rows))
	for _, it := range rows {
		v := VarOutput{Key: it.Key, IsSecret: it.IsSecret, Deleted: it.Deleted}
		if !it.Deleted {
			aad := crypto.ItemAAD(project.ID.String(), envAAD, config.ID.String(), it.Key)
			pt, err := cipher.Open(it.ValueCiphertext, aad)
			if err != nil {
				return nil, err
			}
			v.Value = string(pt)
		}
		out = append(out, v)
	}
	return out, nil
}

// layer resolves an environment slug to (envID pointer, AAD env string). An
// empty or "base" slug means the base layer (nil, "").
func (s *Service) layer(ctx context.Context, projectID uuid.UUID, envSlug string) (*uuid.UUID, string, error) {
	if envSlug == "" || envSlug == "base" {
		return nil, "", nil
	}
	env, err := s.store.Q().GetEnvironmentBySlug(ctx, projectID, envSlug)
	if err != nil {
		if err == store.ErrNotFound {
			return nil, "", invalid("env", "unknown environment: "+envSlug)
		}
		return nil, "", err
	}
	id := env.ID
	return &id, id.String(), nil
}
