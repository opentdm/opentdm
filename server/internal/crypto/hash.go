package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"io"

	"golang.org/x/crypto/hkdf"
)

// TokenHash returns HMAC-SHA256(pepper, token). Service and session tokens are
// high-entropy random secrets, so a fast keyed hash is correct (argon2 on the
// auth hot path would be a self-inflicted DoS). The pepper lives in the
// environment, not the database, so a DB dump alone cannot verify tokens.
func TokenHash(pepper, token []byte) []byte {
	mac := hmac.New(sha256.New, pepper)
	mac.Write(token)
	return mac.Sum(nil)
}

// ConstantTimeEqual reports whether a and b are equal without leaking timing.
func ConstantTimeEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// contentHMACInfo is the HKDF context that derives a per-DEK MAC key distinct
// from the encryption use of the DEK.
const contentHMACInfo = "opentdm-hmac-v1"

// ContentHMAC returns a keyed hash of plaintext for change-detection and dedupe
// without decrypting. The MAC key is derived from the project DEK via HKDF, so
// — unlike a raw sha256(plaintext) — it is NOT a confirmation oracle for
// low-entropy secrets: an attacker without the DEK cannot verify a guess.
func ContentHMAC(dek, plaintext []byte) ([]byte, error) {
	macKey := make([]byte, 32)
	r := hkdf.New(sha256.New, dek, nil, []byte(contentHMACInfo))
	if _, err := io.ReadFull(r, macKey); err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, macKey)
	mac.Write(plaintext)
	sum := mac.Sum(nil)
	for i := range macKey { // best-effort zeroize
		macKey[i] = 0
	}
	return sum, nil
}

// RandomBytes returns n cryptographically-random bytes.
func RandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}
