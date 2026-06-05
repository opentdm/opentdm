package httpapi

import (
	"context"

	"github.com/google/uuid"
)

// actorID returns the authenticated user's ID (for audit/version authorship), or nil.
func actorID(ctx context.Context) *uuid.UUID {
	if u, ok := userFrom(ctx); ok {
		id := u.ID
		return &id
	}
	return nil
}

// strPtr returns nil for an empty string, else a pointer to it.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
