package app

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// NeedsSetup reports whether no users exist yet (first-run).
func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.store.Q().UserCount(ctx)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

// Bootstrap creates the first admin. It requires the one-time setup token
// (printed to logs at first boot) and is guarded by a DB singleton so a second
// concurrent caller fails.
func (s *Service) Bootstrap(ctx context.Context, setupToken, username, email, password string) (model.User, error) {
	s.mu.Lock()
	expected := s.setupToken
	s.mu.Unlock()
	if expected == "" {
		return model.User{}, ErrConflict // already initialized
	}
	if !crypto.ConstantTimeEqual([]byte(setupToken), []byte(expected)) {
		return model.User{}, ErrUnauthorized
	}
	if err := validateCredentials(username, email, password); err != nil {
		return model.User{}, err
	}
	hash, err := crypto.HashPassword(password)
	if err != nil {
		return model.User{}, err
	}

	var created model.User
	err = s.store.InTx(ctx, func(q *store.Queries) error {
		if err := q.ClaimBootstrap(ctx); err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		u, err := q.CreateUser(ctx, model.User{
			Username: username, Email: email, PasswordHash: hash, IsAdmin: true,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return ErrConflict
			}
			return err
		}
		created = u
		return nil
	})
	if err != nil {
		return model.User{}, err
	}
	// Disable further bootstrap.
	s.mu.Lock()
	s.setupToken = ""
	s.mu.Unlock()
	return created, nil
}

// Login verifies credentials and returns a raw session token to set as a cookie.
func (s *Service) Login(ctx context.Context, username, password string) (string, model.User, error) {
	u, err := s.store.Q().GetUserByUsername(ctx, username)
	if err != nil {
		if err == store.ErrNotFound {
			return "", model.User{}, ErrUnauthorized
		}
		return "", model.User{}, err
	}
	ok, err := crypto.VerifyPassword(u.PasswordHash, password)
	if err != nil || !ok {
		return "", model.User{}, ErrUnauthorized
	}
	if !u.IsActive {
		return "", model.User{}, ErrForbidden
	}
	raw, err := newOpaqueToken()
	if err != nil {
		return "", model.User{}, err
	}
	if _, err := s.store.Q().CreateSession(ctx, u.ID, s.hashToken(raw), now().Add(sessionTTL)); err != nil {
		return "", model.User{}, err
	}
	return raw, u, nil
}

// AuthenticateSession resolves a raw session cookie value to its user.
func (s *Service) AuthenticateSession(ctx context.Context, raw string) (model.User, error) {
	if raw == "" {
		return model.User{}, ErrUnauthorized
	}
	u, err := s.store.Q().GetUserBySessionHash(ctx, s.hashToken(raw))
	if err != nil {
		if err == store.ErrNotFound {
			return model.User{}, ErrUnauthorized
		}
		return model.User{}, err
	}
	return u, nil
}

// Logout revokes the session identified by the raw cookie value.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.store.Q().RevokeSession(ctx, s.hashToken(raw))
}

func (s *Service) hashToken(raw string) []byte {
	return crypto.TokenHash(s.pepper, []byte(raw))
}

// newOpaqueToken returns a URL-safe random 32-byte token.
func newOpaqueToken() (string, error) {
	b, err := crypto.RandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// UpdateProfile changes the signed-in user's email (the only editable profile
// field; username is the stable audit/login identity). Duplicate email → 409.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, email string) (model.User, error) {
	email = strings.TrimSpace(email)
	if !strings.Contains(email, "@") {
		return model.User{}, invalid("email", "must be a valid email")
	}
	u, err := s.store.Q().UpdateUserEmail(ctx, userID, email)
	if err != nil {
		if isUniqueViolation(err) {
			return model.User{}, ErrConflict
		}
		return model.User{}, err
	}
	return u, nil
}

// ChangePassword verifies the current password (constant-time) and sets a new
// one. A wrong current password is ErrUnauthorized.
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) error {
	u, err := s.store.Q().GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	ok, err := crypto.VerifyPassword(u.PasswordHash, current)
	if err != nil || !ok {
		return ErrUnauthorized
	}
	if len(next) < 8 {
		return invalid("new_password", "must be at least 8 characters")
	}
	hash, err := crypto.HashPassword(next)
	if err != nil {
		return err
	}
	return s.store.Q().UpdateUserPassword(ctx, userID, hash)
}

func validateCredentials(username, email, password string) error {
	if strings.TrimSpace(username) == "" {
		return invalid("username", "must not be empty")
	}
	if !strings.Contains(email, "@") {
		return invalid("email", "must be a valid email")
	}
	if len(password) < 8 {
		return invalid("password", "must be at least 8 characters")
	}
	return nil
}
