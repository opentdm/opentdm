package app

import (
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

// SetItems encrypts and replaces all items at one (config, layer). envSlug "" or
// "base" targets the base layer.
func (s *Service) SetItems(ctx context.Context, project model.Project, config model.Config, envSlug string, inputs []VarInput) error {
	envID, envAAD, err := s.layer(ctx, project.ID, envSlug)
	if err != nil {
		return err
	}
	cipher, err := s.cipherFor(project)
	if err != nil {
		return err
	}

	items := make([]store.ItemInput, 0, len(inputs))
	for _, in := range inputs {
		if !codec.ValidKey(in.Key) {
			return invalid("key", "invalid variable name: "+in.Key)
		}
		plaintext := in.Value
		if in.Deleted {
			plaintext = "" // tombstone carries no value
		}
		aad := crypto.ItemAAD(project.ID.String(), envAAD, config.ID.String(), in.Key)
		ct, err := cipher.Seal([]byte(plaintext), aad)
		if err != nil {
			return err
		}
		items = append(items, store.ItemInput{
			Key: in.Key, ValueCiphertext: ct, DEKVersion: project.DEKVersion,
			IsSecret: in.IsSecret, Deleted: in.Deleted,
		})
	}
	return s.store.InTx(ctx, func(q *store.Queries) error {
		return q.ReplaceItems(ctx, config.ID, envID, items, nil)
	})
}

// GetItems returns decrypted items at one (config, layer) for the editor.
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
