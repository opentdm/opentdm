package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/resolve"
)

// Resolve decrypts and merges all variable configs for a project at an
// environment, returning the merged result (base → env override, cross-config
// precedence by sort_order, with collisions reported).
func (s *Service) Resolve(ctx context.Context, project model.Project, envID uuid.UUID) (resolve.Result, error) {
	items, err := s.store.Q().ResolveItems(ctx, project.ID, envID)
	if err != nil {
		return resolve.Result{}, err
	}
	cipher, err := s.cipherFor(project)
	if err != nil {
		return resolve.Result{}, err
	}

	// Group items per config, preserving sort_order.
	type group struct {
		input resolve.ConfigInput
	}
	order := []uuid.UUID{}
	byConfig := map[uuid.UUID]*group{}

	for _, it := range items {
		g, ok := byConfig[it.ConfigID]
		if !ok {
			g = &group{input: resolve.ConfigInput{ConfigName: it.ConfigName, SortOrder: it.SortOrder}}
			byConfig[it.ConfigID] = g
			order = append(order, it.ConfigID)
		}
		v := resolve.Variable{Key: it.Key, IsSecret: it.IsSecret, Deleted: it.Deleted}
		if !it.Deleted {
			envAAD := ""
			if !it.IsBase {
				envAAD = envID.String()
			}
			aad := crypto.ItemAAD(project.ID.String(), envAAD, it.ConfigID.String(), it.Key)
			pt, err := cipher.Open(it.ValueCiphertext, aad)
			if err != nil {
				return resolve.Result{}, err
			}
			v.Value = string(pt)
		}
		if it.IsBase {
			g.input.Base = append(g.input.Base, v)
		} else {
			g.input.Override = append(g.input.Override, v)
		}
	}

	configs := make([]resolve.ConfigInput, 0, len(order))
	for _, id := range order {
		configs = append(configs, byConfig[id].input)
	}
	return resolve.Merge(configs), nil
}

// ResolveConfig decrypts and merges a SINGLE variable config at an environment
// (base → env override, with tombstones), returning its resolved variables.
// Unlike Resolve there is no cross-config merge, so the result never carries
// collisions. File configs are rejected (their per-env content is fetched via
// the blob path, not resolved).
func (s *Service) ResolveConfig(ctx context.Context, project model.Project, config model.Config, envID uuid.UUID) (resolve.Result, error) {
	if config.Kind != model.KindVariable {
		return resolve.Result{}, invalid("kind", "resolve is only supported for variable configs")
	}
	items, err := s.store.Q().ResolveItemsForConfig(ctx, config.ID, envID)
	if err != nil {
		return resolve.Result{}, err
	}
	cipher, err := s.cipherFor(project)
	if err != nil {
		return resolve.Result{}, err
	}

	input := resolve.ConfigInput{ConfigName: config.Name, SortOrder: config.SortOrder}
	for _, it := range items {
		v := resolve.Variable{Key: it.Key, IsSecret: it.IsSecret, Deleted: it.Deleted}
		if !it.Deleted {
			envAAD := ""
			if !it.IsBase {
				envAAD = envID.String()
			}
			aad := crypto.ItemAAD(project.ID.String(), envAAD, config.ID.String(), it.Key)
			pt, err := cipher.Open(it.ValueCiphertext, aad)
			if err != nil {
				return resolve.Result{}, err
			}
			v.Value = string(pt)
		}
		if it.IsBase {
			input.Base = append(input.Base, v)
		} else {
			input.Override = append(input.Override, v)
		}
	}
	return resolve.ResolveOne(input), nil
}
