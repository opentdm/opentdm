package app

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// Cloning copies an object's content from one environment layer to another.
//
// Values are encrypted with AAD that binds the environment (see crypto.ItemAAD /
// crypto.BlobAAD), so a clone is NOT a ciphertext row-copy: it decrypts under the
// SOURCE env's AAD and re-encrypts under the TARGET env's AAD. The existing
// GetItems/SetItems and GetBlob/SetBlob already do exactly that split (and handle
// validation, versioning, tombstones, and the version-race 409), so clone is a
// thin orchestration over them — mirroring Service.Rollback.
//
// Semantics: clone copies an env's OWN override layer (what the editor shows),
// not the base-merged view. Since every env shares the base, this gives the
// target the same resolved values as the source while preserving inheritance.

// CloneFailure records one config that could not be cloned during a bulk clone.
type CloneFailure struct {
	Config string
	Reason string
}

// CloneSummary is the result of cloning a whole environment. It carries config
// NAMES and counts only — never any values.
type CloneSummary struct {
	Cloned    []string
	Unchanged []string
	Skipped   []string
	Failed    []CloneFailure
}

// CloneLayer clones a single config's fromSlug layer into toSlug. For variable
// configs, withValues=false copies the keys with empty values (which override —
// hide — inherited base values until filled in). For file configs withValues is
// ignored (content is always copied whole).
func (s *Service) CloneLayer(ctx context.Context, project model.Project, config model.Config, fromSlug, toSlug string, withValues bool, actor *uuid.UUID) (model.ConfigVersion, error) {
	// Validate both layers exist before the self-clone check, so an unknown env
	// reports "unknown environment" rather than a misleading "onto itself".
	fromEnvID, fromAAD, err := s.layer(ctx, project.ID, fromSlug)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	if _, _, err := s.layer(ctx, project.ID, toSlug); err != nil {
		return model.ConfigVersion{}, err
	}
	if normLayer(fromSlug) == normLayer(toSlug) {
		return model.ConfigVersion{}, invalid("to", "cannot clone an environment onto itself")
	}
	version, hadSource, err := s.cloneConfigInto(ctx, project, config, fromSlug, toSlug, fromEnvID, fromAAD, withValues, actor)
	if err != nil {
		return model.ConfigVersion{}, err
	}
	if !hadSource {
		return model.ConfigVersion{}, invalid("from", "source layer "+normLayer(fromSlug)+" has nothing to clone")
	}
	return version, nil
}

// CloneEnvironment clones every non-archived config's fromSlug layer into toSlug.
// Each config is its own transaction/version (not globally atomic); configs whose
// source layer is empty are skipped, and per-config failures are reported rather
// than aborting the batch.
func (s *Service) CloneEnvironment(ctx context.Context, project model.Project, fromSlug, toSlug string, withValues bool, actor *uuid.UUID) (CloneSummary, error) {
	fromEnvID, fromAAD, err := s.layer(ctx, project.ID, fromSlug)
	if err != nil {
		return CloneSummary{}, err
	}
	toEnvID, _, err := s.layer(ctx, project.ID, toSlug)
	if err != nil {
		return CloneSummary{}, err
	}
	if normLayer(fromSlug) == normLayer(toSlug) {
		return CloneSummary{}, invalid("to", "cannot clone an environment onto itself")
	}
	configs, err := s.ListConfigs(ctx, project.ID) // excludes archived
	if err != nil {
		return CloneSummary{}, err
	}

	var sum CloneSummary
	for _, c := range configs {
		// Baseline the target's current version so a dedup no-op can be reported as
		// "unchanged". A missing layer (ErrNotFound) means version 0; any other read
		// error is a real failure for this config, not a silent 0.
		priorN := 0
		cur, err := s.store.Q().GetCurrentVersion(ctx, c.ID, toEnvID)
		if err == nil {
			priorN = cur.Version
		} else if !errors.Is(err, store.ErrNotFound) {
			sum.Failed = append(sum.Failed, CloneFailure{Config: c.Name, Reason: cloneReason(err)})
			continue
		}
		version, hadSource, err := s.cloneConfigInto(ctx, project, c, fromSlug, toSlug, fromEnvID, fromAAD, withValues, actor)
		switch {
		case err != nil:
			sum.Failed = append(sum.Failed, CloneFailure{Config: c.Name, Reason: cloneReason(err)})
		case !hadSource:
			sum.Skipped = append(sum.Skipped, c.Name)
		case version.Version == priorN:
			sum.Unchanged = append(sum.Unchanged, c.Name) // dedup no-op: content was identical
		default:
			sum.Cloned = append(sum.Cloned, c.Name)
		}
	}
	return sum, nil
}

// cloneConfigInto clones one config's source layer into the target layer. It
// returns (version, hadSource, error): hadSource=false means the source layer is
// empty (no variable rows / no override blob) so the caller should skip it. The
// re-encryption under the target env happens inside SetItems/SetBlob.
func (s *Service) cloneConfigInto(ctx context.Context, project model.Project, config model.Config, fromSlug, toSlug string, fromEnvID *uuid.UUID, fromAAD string, withValues bool, actor *uuid.UUID) (model.ConfigVersion, bool, error) {
	comment := "cloned from " + normLayer(fromSlug)

	if config.Kind == model.KindFile {
		// Read the SOURCE env's own override blob (no base fallback — pair the AAD
		// to the exact row read). An absent override means the source inherits base,
		// just as the target already would, so there is nothing to clone.
		blob, err := s.store.Q().GetBlob(ctx, config.ID, fromEnvID)
		if err == store.ErrNotFound {
			return model.ConfigVersion{}, false, nil
		}
		if err != nil {
			return model.ConfigVersion{}, false, err
		}
		cipher, err := s.cipherFor(project)
		if err != nil {
			return model.ConfigVersion{}, false, err
		}
		content, err := cipher.Open(blob.ContentCiphertext, crypto.BlobAAD(project.ID.String(), fromAAD, config.ID.String()))
		if err != nil {
			return model.ConfigVersion{}, false, err
		}
		// maxBytes=0: the source content already passed the size cap on write
		// (mirrors Rollback).
		version, err := s.SetBlob(ctx, project, config, toSlug, content, 0, &comment, actor)
		return version, true, err
	}

	// Variable config: decrypt the source override layer (incl. tombstones) and
	// re-persist it into the target layer.
	src, err := s.GetItems(ctx, project, config, fromSlug)
	if err != nil {
		return model.ConfigVersion{}, false, err
	}
	if len(src) == 0 {
		return model.ConfigVersion{}, false, nil
	}
	inputs := make([]VarInput, 0, len(src))
	for _, it := range src {
		in := VarInput{Key: it.Key, IsSecret: it.IsSecret, Deleted: it.Deleted}
		if withValues {
			in.Value = it.Value
		}
		inputs = append(inputs, in)
	}
	version, err := s.SetItems(ctx, project, config, toSlug, inputs, &comment, actor)
	return version, true, err
}

// normLayer canonicalizes the base layer so "" and "base" compare equal.
func normLayer(slug string) string {
	if slug == "" {
		return "base"
	}
	return slug
}

// cloneReason renders a domain error as a short, value-free reason for a bulk
// clone summary.
func cloneReason(err error) string {
	if errors.Is(err, ErrConflict) {
		return "archived or changed concurrently"
	}
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve.Msg
	}
	return "internal error"
}
