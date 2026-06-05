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

// patPrefix distinguishes user PATs from service tokens (tokenPrefix "otdm_").
// They are mutually non-prefixing (5th byte '_' vs 'u'), so HasPrefix dispatch
// is unambiguous.
const patPrefix = "otdmu_"

// MintPAT creates a user-scoped Personal Access Token and returns the raw secret
// (shown once) plus the stored record.
func (s *Service) MintPAT(ctx context.Context, user model.User, name string, expiresAt *time.Time) (string, model.UserPAT, error) {
	if name == "" {
		return "", model.UserPAT{}, invalid("name", "must not be empty")
	}
	secret, err := crypto.RandomBytes(32)
	if err != nil {
		return "", model.UserPAT{}, err
	}
	raw := patPrefix + base64.RawURLEncoding.EncodeToString(secret)
	hash := crypto.TokenHash(s.pepper, []byte(raw))
	prefix := raw
	if len(prefix) > 14 {
		prefix = raw[:14] // "otdmu_" + 8 chars, for display
	}
	created, err := s.store.Q().CreateUserPAT(ctx, model.UserPAT{
		UserID: user.ID, Name: name, Prefix: prefix, ExpiresAt: expiresAt,
	}, hash)
	if err != nil {
		if isUniqueViolation(err) {
			return "", model.UserPAT{}, ErrConflict
		}
		return "", model.UserPAT{}, err
	}
	return raw, created, nil
}

// AuthenticatePAT resolves a raw PAT to its user, rejecting revoked/expired PATs
// and inactive users (all enforced in SQL).
func (s *Service) AuthenticatePAT(ctx context.Context, raw string) (model.User, error) {
	if !strings.HasPrefix(raw, patPrefix) {
		return model.User{}, ErrUnauthorized
	}
	hash := crypto.TokenHash(s.pepper, []byte(raw))
	user, patID, err := s.store.Q().GetUserByPATHash(ctx, hash)
	if err != nil {
		if err == store.ErrNotFound {
			return model.User{}, ErrUnauthorized
		}
		return model.User{}, err
	}
	_ = s.store.Q().TouchUserPAT(ctx, patID, now())
	return user, nil
}

func (s *Service) ListPATs(ctx context.Context, userID uuid.UUID) ([]model.UserPAT, error) {
	return s.store.Q().ListUserPATs(ctx, userID)
}

func (s *Service) RevokePAT(ctx context.Context, userID, patID uuid.UUID) error {
	return s.store.Q().RevokeUserPAT(ctx, userID, patID)
}
