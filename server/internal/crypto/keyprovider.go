// Package crypto implements opentdm's server-side envelope encryption:
//
//	master key (KEK) -> per-project DEK (wrapped) -> per-value/blob ciphertext
//
// The master key is supplied by a KeyProvider (env var in v1; KMS later). DEKs
// are wrapped by the KEK and stored in the database; values/blobs are sealed
// with the project DEK. See DECISIONS.md for the binding rules.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// KeyProvider wraps and unwraps per-project Data Encryption Keys (DEKs) using a
// Key Encryption Key (the master key / KEK). Implementations: EnvKeyProvider
// (v1) and, later, AWS/GCP KMS behind this same interface.
type KeyProvider interface {
	// Wrap encrypts a plaintext DEK with the active KEK. It returns the wrapped
	// blob and the keyRef identifying which KEK was used (for rotation).
	Wrap(plaintextDEK []byte) (wrapped []byte, keyRef string, err error)
	// Unwrap decrypts a wrapped DEK using the KEK identified by keyRef.
	Unwrap(wrapped []byte, keyRef string) (plaintextDEK []byte, err error)
	// ActiveKeyRef is the keyRef new DEKs are wrapped under.
	ActiveKeyRef() string
}

// EnvKeyProvider wraps DEKs locally with AES-256-GCM using master keys supplied
// from the environment. The active key wraps new DEKs; older keys are retained
// (keyed by ref) so existing DEKs can still be unwrapped during a rotation.
type EnvKeyProvider struct {
	activeRef string
	aeads     map[string]cipher.AEAD
}

// NewEnvKeyProvider builds a provider whose active KEK is activeKey (32 bytes),
// referenced as activeRef. olderKeys maps a keyRef to a retired 32-byte KEK that
// must still be able to unwrap DEKs wrapped under it.
func NewEnvKeyProvider(activeRef string, activeKey []byte, olderKeys map[string][]byte) (*EnvKeyProvider, error) {
	if activeRef == "" {
		return nil, errors.New("crypto: active key ref must not be empty")
	}
	p := &EnvKeyProvider{activeRef: activeRef, aeads: make(map[string]cipher.AEAD)}
	if err := p.add(activeRef, activeKey); err != nil {
		return nil, fmt.Errorf("crypto: active key: %w", err)
	}
	for ref, k := range olderKeys {
		if err := p.add(ref, k); err != nil {
			return nil, fmt.Errorf("crypto: old key %q: %w", ref, err)
		}
	}
	return p, nil
}

func (p *EnvKeyProvider) add(ref string, key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("master key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	p.aeads[ref] = aead
	return nil
}

func (p *EnvKeyProvider) ActiveKeyRef() string { return p.activeRef }

// Wrap seals the DEK under the active KEK, binding the keyRef as additional
// authenticated data so a swapped keyRef fails authentication (blocks a
// key-downgrade attack). Wire format: [12-byte nonce][ciphertext+tag].
func (p *EnvKeyProvider) Wrap(plaintextDEK []byte) ([]byte, string, error) {
	aead := p.aeads[p.activeRef]
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, "", err
	}
	ct := aead.Seal(nil, nonce, plaintextDEK, []byte(p.activeRef))
	return append(nonce, ct...), p.activeRef, nil
}

// Unwrap reverses Wrap using the KEK identified by keyRef.
func (p *EnvKeyProvider) Unwrap(wrapped []byte, keyRef string) ([]byte, error) {
	aead, ok := p.aeads[keyRef]
	if !ok {
		return nil, fmt.Errorf("crypto: no master key for ref %q", keyRef)
	}
	ns := aead.NonceSize()
	if len(wrapped) < ns {
		return nil, errors.New("crypto: wrapped DEK too short")
	}
	nonce, ct := wrapped[:ns], wrapped[ns:]
	dek, err := aead.Open(nil, nonce, ct, []byte(keyRef))
	if err != nil {
		return nil, fmt.Errorf("crypto: unwrap DEK: %w", err)
	}
	return dek, nil
}

// NewDEK returns a fresh random 32-byte Data Encryption Key.
func NewDEK() ([]byte, error) {
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, err
	}
	return dek, nil
}
