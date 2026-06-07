package app

import (
	"testing"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

// TokenAllowsEnv is the default-deny scope gate enforced on every /resolve. A
// regression to default-allow would silently widen every token's reach, so pin
// the behavior with a table.
func TestTokenAllowsEnv(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name   string
		envIDs []uuid.UUID
		query  uuid.UUID
		want   bool
	}{
		{"empty scope denies (default-deny)", nil, a, false},
		{"single match allows", []uuid.UUID{a}, a, true},
		{"single non-match denies", []uuid.UUID{a}, b, false},
		{"multi-env match allows", []uuid.UUID{a, b}, b, true},
		{"multi-env non-match denies", []uuid.UUID{a, b}, c, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := model.Token{EnvIDs: tt.envIDs}
			if got := TokenAllowsEnv(tok, tt.query); got != tt.want {
				t.Fatalf("TokenAllowsEnv(envIDs=%v, %v) = %v, want %v", tt.envIDs, tt.query, got, tt.want)
			}
		})
	}
}
