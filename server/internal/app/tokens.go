package app

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// tokenPrefix is the scanner-registerable prefix for service tokens.
const tokenPrefix = "otdm_"

// MintToken creates a project+environment-scoped service token and returns the
// raw secret (shown once) plus the stored record. envSlugs must be non-empty
// (default-deny: a token with no environments can read nothing).
func (s *Service) MintToken(ctx context.Context, creator model.User, project model.Project, name, scope string, envSlugs []string, expiresAt *time.Time) (string, model.Token, error) {
	if name == "" {
		return "", model.Token{}, invalid("name", "must not be empty")
	}
	if scope != model.ScopeRead && scope != model.ScopeWrite {
		return "", model.Token{}, invalid("scope", "must be read or write")
	}
	if len(envSlugs) == 0 {
		return "", model.Token{}, invalid("environments", "a token must be scoped to at least one environment")
	}
	envIDs := make([]uuid.UUID, 0, len(envSlugs))
	for _, slug := range envSlugs {
		env, err := s.store.Q().GetEnvironmentBySlug(ctx, project.ID, slug)
		if err != nil {
			if err == store.ErrNotFound {
				return "", model.Token{}, invalid("environments", "unknown environment: "+slug)
			}
			return "", model.Token{}, err
		}
		envIDs = append(envIDs, env.ID)
	}

	secret, err := crypto.RandomBytes(32)
	if err != nil {
		return "", model.Token{}, err
	}
	raw := tokenPrefix + base64.RawURLEncoding.EncodeToString(secret)
	hash := crypto.TokenHash(s.pepper, []byte(raw))
	prefix := raw
	if len(prefix) > 13 {
		prefix = raw[:13] // "otdm_" + 8 chars, for display
	}

	creatorID := creator.ID
	var created model.Token
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		t, err := q.CreateToken(ctx, model.Token{
			ProjectID: project.ID, Name: name, Prefix: prefix, Scope: scope,
			EnvIDs: envIDs, ExpiresAt: expiresAt,
		}, hash, &creatorID)
		if err != nil {
			return err
		}
		created = t
		return nil
	})
	if err != nil {
		return "", model.Token{}, err
	}
	return raw, created, nil
}

// AuthenticateToken resolves a raw service token to its record, rejecting
// revoked or expired tokens.
func (s *Service) AuthenticateToken(ctx context.Context, raw string) (model.Token, error) {
	if !strings.HasPrefix(raw, tokenPrefix) {
		return model.Token{}, ErrUnauthorized
	}
	hash := crypto.TokenHash(s.pepper, []byte(raw))
	t, err := s.store.Q().GetTokenByHash(ctx, hash)
	if err != nil {
		if err == store.ErrNotFound {
			return model.Token{}, ErrUnauthorized
		}
		return model.Token{}, err
	}
	if t.RevokedAt != nil {
		return model.Token{}, ErrUnauthorized
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(now()) {
		return model.Token{}, ErrUnauthorized
	}
	// Last-used is coalesced in memory and flushed on an interval (see touch.go),
	// so the hot /resolve path never issues a per-request UPDATE.
	s.recordTokenUse(t.ID)
	return t, nil
}

// TokenAllowsEnv reports whether a token is scoped to the given environment
// (default-deny).
func TokenAllowsEnv(t model.Token, envID uuid.UUID) bool {
	for _, id := range t.EnvIDs {
		if id == envID {
			return true
		}
	}
	return false
}

func (s *Service) ListTokens(ctx context.Context, projectID uuid.UUID) ([]model.Token, error) {
	return s.store.Q().ListTokens(ctx, projectID)
}

func (s *Service) RevokeToken(ctx context.Context, projectID, tokenID uuid.UUID) error {
	return s.store.Q().RevokeToken(ctx, projectID, tokenID)
}
