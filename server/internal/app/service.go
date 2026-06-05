// Package app is opentdm's service layer: it orchestrates the store and crypto
// to implement use cases (bootstrap, login, projects, configs, resolve, tokens).
// HTTP handlers call these methods; business rules live here, not in handlers.
package app

import (
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/model"
	"github.com/opentdm/opentdm/server/internal/store"
)

// Sentinel errors, mapped to HTTP status by the handler layer.
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrNotFound     = store.ErrNotFound
)

// ValidationError is a 4xx caused by bad input.
type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Msg }

func invalid(field, msg string) error { return &ValidationError{Field: field, Msg: msg} }

const sessionTTL = 30 * 24 * time.Hour

// now is a package clock (overridable in tests).
var now = time.Now

// Service holds dependencies for the use cases.
type Service struct {
	store  *store.Store
	keys   crypto.KeyProvider
	pepper []byte

	mu         sync.Mutex // guards setupToken
	setupToken string     // one-time first-boot admin token (empty once a user exists)

	cipherCache sync.Map // project ID string -> *crypto.DEKCipher
}

// NewService constructs the service. setupToken is the one-time bootstrap token
// (printed to logs at first boot); pass "" if a user already exists.
func NewService(st *store.Store, keys crypto.KeyProvider, pepper []byte, setupToken string) *Service {
	return &Service{store: st, keys: keys, pepper: pepper, setupToken: setupToken}
}

// cipherFor returns a DEKCipher for a project, unwrapping and caching its DEK.
func (s *Service) cipherFor(p model.Project) (*crypto.DEKCipher, error) {
	key := p.ID.String()
	if c, ok := s.cipherCache.Load(key); ok {
		return c.(*crypto.DEKCipher), nil
	}
	dek, err := s.keys.Unwrap(p.DEKWrapped, p.DEKKeyRef)
	if err != nil {
		return nil, err
	}
	c, err := crypto.NewDEKCipher(dek, crypto.AlgAESGCM)
	zero(dek)
	if err != nil {
		return nil, err
	}
	actual, _ := s.cipherCache.LoadOrStore(key, c)
	return actual.(*crypto.DEKCipher), nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
