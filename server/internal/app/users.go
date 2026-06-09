package app

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

// validColorModes are the accepted UI color-mode preference values.
var validColorModes = map[string]bool{"": true, "light": true, "dark": true, "auto": true}

// ListUsers returns all users (admin directory).
func (s *Service) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.store.Q().ListUsers(ctx)
}

// UpdateUser sets a user's is_active / is_admin flags (nil leaves unchanged),
// refusing a change that would remove the last active instance admin (so the
// instance can't be locked out).
func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, isActive, isAdmin *bool) (model.User, error) {
	demotes := isAdmin != nil && !*isAdmin
	deactivates := isActive != nil && !*isActive
	if demotes || deactivates {
		target, err := s.store.Q().GetUserByID(ctx, id)
		if err != nil {
			return model.User{}, err
		}
		if target.IsAdmin && target.IsActive {
			n, err := s.store.Q().CountActiveAdmins(ctx)
			if err != nil {
				return model.User{}, err
			}
			if n <= 1 {
				return model.User{}, invalid("user", "cannot remove the last active admin")
			}
		}
	}
	return s.store.Q().UpdateUserFlags(ctx, id, isActive, isAdmin)
}

// UpdatePreferences validates and stores a user's UI preferences (theme +
// favourite project slugs). Re-marshaling to a canonical blob strips any
// unknown fields the client may have sent.
func (s *Service) UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs model.UserPreferences) (model.User, error) {
	if !validColorModes[prefs.ColorMode] {
		return model.User{}, invalid("color_mode", "must be light, dark, or auto")
	}
	if len(prefs.Favourites) > 500 {
		return model.User{}, invalid("favourites", "too many favourites")
	}
	raw, err := json.Marshal(prefs)
	if err != nil {
		return model.User{}, err
	}
	return s.store.Q().UpdatePreferences(ctx, userID, raw)
}
