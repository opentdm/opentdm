package app

import (
	"bytes"
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/codec"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// likeEscaper neutralizes ILIKE wildcards in user-supplied search text so the
// query is a literal substring match (ESCAPE '\' in the SQL).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// SearchConfigs returns up to 20 objects whose name matches query across the
// projects the user can access. Empty query → no results.
func (s *Service) SearchConfigs(ctx context.Context, user model.User, query string) ([]model.ConfigSearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	return s.store.Q().SearchConfigs(ctx, user.ID, user.IsAdmin, likeEscaper.Replace(query), 20)
}

var variableFormats = map[string]bool{
	model.FormatEnv: true, model.FormatProperties: true, model.FormatSecret: true,
}
var fileFormats = map[string]bool{
	model.FormatJSON: true, model.FormatCSV: true, model.FormatXML: true, model.FormatYAML: true,
}

// envOnlyMode gates config CREATION to variable/env configs. It is a reversible
// product flag — false re-enables properties/secret and file (json/csv/xml)
// configs. Re-opened for the file-upload create flow (the web Add-object
// dropzone): .env uploads create variable/env bundles, json/csv/xml uploads
// create file configs. The format-enum maps and the DB CHECK constraint are the
// real guards; this flag only narrows what the create endpoint accepts.
const envOnlyMode = false

// CreateConfig creates a config after validating the kind/format pairing.
func (s *Service) CreateConfig(ctx context.Context, creator model.User, projectID uuid.UUID, c model.Config) (model.Config, error) {
	if c.Name == "" {
		return model.Config{}, invalid("name", "must not be empty")
	}
	if envOnlyMode && (c.Kind != model.KindVariable || c.Format != model.FormatEnv) {
		return model.Config{}, invalid("format", "only env configs can be created")
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

// UpdateConfig renames a config and sets its sort_order/description.
func (s *Service) UpdateConfig(ctx context.Context, projectID, configID uuid.UUID, name string, sortOrder int, description string) (model.Config, error) {
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
		c, err := q.UpdateConfig(ctx, model.Config{ID: configID, Name: name, SortOrder: sortOrder, Description: description})
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
//
// Inheritance convenience: when writing a NON-base layer, every new key (one not
// yet present in base) is also seeded into base with an empty value in the same
// transaction — so a variable added in one environment "exists everywhere"
// (inherited as empty elsewhere). Tombstones never propagate.
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

	var version model.ConfigVersion
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		version, err = s.writeLayer(ctx, q, project, config, envID, envAAD, cipher, dek, inputs, comment, actor)
		if err != nil {
			return err
		}
		if envID != nil {
			baseInputs, err := s.baseWithSeededKeys(ctx, q, project, config, cipher, inputs)
			if err != nil {
				return err
			}
			if baseInputs != nil {
				if _, err := s.writeLayer(ctx, q, project, config, nil, "", cipher, dek, baseInputs, nil, actor); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return model.ConfigVersion{}, ErrConflict // concurrent save lost the version-number race
		}
		return model.ConfigVersion{}, err
	}
	return version, nil
}

// writeLayer encrypts the inputs and replaces all items at one (config, layer),
// then appends a version snapshot — within the caller's transaction.
func (s *Service) writeLayer(ctx context.Context, q *store.Queries, project model.Project, config model.Config, envID *uuid.UUID, envAAD string, cipher *crypto.DEKCipher, dek []byte, inputs []VarInput, comment *string, actor *uuid.UUID) (model.ConfigVersion, error) {
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
	if err := q.ReplaceItems(ctx, config.ID, envID, items, actor); err != nil {
		return model.ConfigVersion{}, err
	}
	return s.appendVersionDedup(ctx, q, config.ID, envID, hmac, store.VersionInput{
		SnapshotKind: model.KindVariable, SnapshotCiphertext: snapCT, DEKVersion: project.DEKVersion,
		ContentHMAC: hmac, ByteSize: int64(len(canon)), Comment: comment, CreatedBy: actor,
	})
}

// baseWithSeededKeys returns the full base-layer inputs (current base, decrypted,
// plus any new keys from an env-layer write seeded with an empty value) when at
// least one new key would be added; otherwise nil to signal "base unchanged, skip
// the write". Runs inside the caller's transaction.
func (s *Service) baseWithSeededKeys(ctx context.Context, q *store.Queries, project model.Project, config model.Config, cipher *crypto.DEKCipher, envInputs []VarInput) ([]VarInput, error) {
	baseRows, err := q.ListItems(ctx, config.ID, nil)
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(baseRows))
	out := make([]VarInput, 0, len(baseRows)+1)
	for _, it := range baseRows {
		have[it.Key] = true
		v := VarInput{Key: it.Key, IsSecret: it.IsSecret, Deleted: it.Deleted}
		if !it.Deleted {
			pt, err := cipher.Open(it.ValueCiphertext, crypto.ItemAAD(project.ID.String(), "", config.ID.String(), it.Key))
			if err != nil {
				return nil, err
			}
			v.Value = string(pt)
		}
		out = append(out, v)
	}
	added := false
	for _, in := range envInputs {
		if in.Deleted || have[in.Key] {
			continue
		}
		out = append(out, VarInput{Key: in.Key}) // empty value, not secret
		have[in.Key] = true
		added = true
	}
	if !added {
		return nil, nil
	}
	return out, nil
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
