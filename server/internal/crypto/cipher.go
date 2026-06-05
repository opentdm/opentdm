package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// Algorithm identifiers for the versioned wire format. The first byte of every
// sealed blob records which AEAD produced it, so the algorithm can evolve (and
// coexist) without ambiguity.
const (
	AlgAESGCM  byte = 0x01 // AES-256-GCM, 12-byte nonce
	AlgXChaCha byte = 0x02 // XChaCha20-Poly1305, 24-byte nonce
)

// Wire format: [1-byte alg][nonce][ciphertext||tag]. The nonce length is
// determined by the algorithm.

// DEKCipher seals and opens values/blobs with a project's Data Encryption Key.
// It can open data sealed by either supported algorithm (both AEADs are built
// from the same 32-byte DEK), and seals new data with sealAlg. This makes a DEK
// algorithm migration a lazy re-encrypt rather than a flag day.
type DEKCipher struct {
	gcm     cipher.AEAD
	xchacha cipher.AEAD
	sealAlg byte
}

// NewDEKCipher builds a cipher from a 32-byte DEK. sealAlg selects which
// algorithm Seal uses (default AlgAESGCM per the locked decision); pass 0 for
// the default.
func NewDEKCipher(dek []byte, sealAlg byte) (*DEKCipher, error) {
	if len(dek) != 32 {
		return nil, fmt.Errorf("crypto: DEK must be 32 bytes, got %d", len(dek))
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	xc, err := chacha20poly1305.NewX(dek)
	if err != nil {
		return nil, err
	}
	if sealAlg == 0 {
		sealAlg = AlgAESGCM
	}
	if sealAlg != AlgAESGCM && sealAlg != AlgXChaCha {
		return nil, fmt.Errorf("crypto: unknown seal algorithm 0x%02x", sealAlg)
	}
	return &DEKCipher{gcm: gcm, xchacha: xc, sealAlg: sealAlg}, nil
}

func (c *DEKCipher) aeadFor(alg byte) (cipher.AEAD, error) {
	switch alg {
	case AlgAESGCM:
		return c.gcm, nil
	case AlgXChaCha:
		return c.xchacha, nil
	default:
		return nil, fmt.Errorf("crypto: unknown algorithm 0x%02x", alg)
	}
}

// Seal encrypts plaintext, authenticating aad (which must bind the row's
// immutable identity — see AAD). A fresh random nonce is generated for every
// call; nonce reuse with the same key is catastrophic for these AEADs and must
// never happen.
func (c *DEKCipher) Seal(plaintext, aad []byte) ([]byte, error) {
	aead, err := c.aeadFor(c.sealAlg)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, 1+len(nonce)+len(plaintext)+aead.Overhead())
	out = append(out, c.sealAlg)
	out = append(out, nonce...)
	out = aead.Seal(out, nonce, plaintext, aad)
	return out, nil
}

// Open decrypts a blob produced by Seal, dispatching on its algorithm byte. It
// fails closed if the blob was tampered with or aad does not match (e.g. a row
// relocated to a different identity).
func (c *DEKCipher) Open(blob, aad []byte) ([]byte, error) {
	if len(blob) < 1 {
		return nil, errors.New("crypto: empty ciphertext")
	}
	alg := blob[0]
	aead, err := c.aeadFor(alg)
	if err != nil {
		return nil, err
	}
	ns := aead.NonceSize()
	if len(blob) < 1+ns {
		return nil, errors.New("crypto: ciphertext too short")
	}
	nonce := blob[1 : 1+ns]
	ct := blob[1+ns:]
	pt, err := aead.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, fmt.Errorf("crypto: open: %w", err)
	}
	return pt, nil
}
