package crypto

import (
	"bytes"
	"testing"
)

func mustDEK(t *testing.T) []byte {
	t.Helper()
	dek, err := NewDEK()
	if err != nil {
		t.Fatalf("NewDEK: %v", err)
	}
	return dek
}

func TestDEKCipher_RoundTrip(t *testing.T) {
	dek := mustDEK(t)
	for _, alg := range []byte{AlgAESGCM, AlgXChaCha} {
		c, err := NewDEKCipher(dek, alg)
		if err != nil {
			t.Fatalf("NewDEKCipher: %v", err)
		}
		plaintext := []byte("postgres://user:pass@host:5432/db")
		aad := ItemAAD("proj", "env", "cfg", "DATABASE_URL")
		blob, err := c.Seal(plaintext, aad)
		if err != nil {
			t.Fatalf("Seal: %v", err)
		}
		if blob[0] != alg {
			t.Fatalf("alg byte = 0x%02x, want 0x%02x", blob[0], alg)
		}
		got, err := c.Open(blob, aad)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Fatalf("round-trip mismatch: got %q", got)
		}
	}
}

func TestDEKCipher_OpenAcrossAlgorithms(t *testing.T) {
	// A cipher whose seal-alg is AES must still open XChaCha-sealed data, because
	// a DEK algorithm migration is lazy. Both AEADs derive from the same DEK.
	dek := mustDEK(t)
	xc, _ := NewDEKCipher(dek, AlgXChaCha)
	aad := ItemAAD("p", "e", "c", "K")
	blob, err := xc.Seal([]byte("value"), aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	gcm, _ := NewDEKCipher(dek, AlgAESGCM)
	got, err := gcm.Open(blob, aad)
	if err != nil {
		t.Fatalf("cross-alg Open: %v", err)
	}
	if string(got) != "value" {
		t.Fatalf("got %q", got)
	}
}

func TestDEKCipher_AADMismatchFailsClosed(t *testing.T) {
	dek := mustDEK(t)
	c, _ := NewDEKCipher(dek, AlgAESGCM)
	blob, _ := c.Seal([]byte("secret"), ItemAAD("p", "staging", "c", "KEY"))
	// Same ciphertext, different identity (env changed) must not open.
	if _, err := c.Open(blob, ItemAAD("p", "production", "c", "KEY")); err == nil {
		t.Fatal("expected Open to fail when AAD identity differs (relocation), got nil")
	}
}

func TestDEKCipher_TamperFailsClosed(t *testing.T) {
	dek := mustDEK(t)
	c, _ := NewDEKCipher(dek, AlgAESGCM)
	aad := ItemAAD("p", "e", "c", "K")
	blob, _ := c.Seal([]byte("secret"), aad)
	blob[len(blob)-1] ^= 0xFF // flip a tag bit
	if _, err := c.Open(blob, aad); err == nil {
		t.Fatal("expected Open to fail on tampered ciphertext")
	}
}

func TestDEKCipher_WrongKeyFails(t *testing.T) {
	aad := ItemAAD("p", "e", "c", "K")
	c1, _ := NewDEKCipher(mustDEK(t), AlgAESGCM)
	blob, _ := c1.Seal([]byte("secret"), aad)
	c2, _ := NewDEKCipher(mustDEK(t), AlgAESGCM)
	if _, err := c2.Open(blob, aad); err == nil {
		t.Fatal("expected Open with a different DEK to fail")
	}
}

// TestDEKCipher_NonceUniqueness is the property test the design calls out:
// nonce reuse with the same key is catastrophic, so verify many seals of the
// same plaintext all use distinct nonces (and produce distinct ciphertexts).
func TestDEKCipher_NonceUniqueness(t *testing.T) {
	dek := mustDEK(t)
	for _, alg := range []byte{AlgAESGCM, AlgXChaCha} {
		c, _ := NewDEKCipher(dek, alg)
		aad := ItemAAD("p", "e", "c", "K")
		const n = 20000
		nonceSize := 12
		if alg == AlgXChaCha {
			nonceSize = 24
		}
		seen := make(map[string]struct{}, n)
		for i := 0; i < n; i++ {
			blob, err := c.Seal([]byte("same-plaintext"), aad)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}
			nonce := string(blob[1 : 1+nonceSize])
			if _, dup := seen[nonce]; dup {
				t.Fatalf("alg 0x%02x: nonce collision after %d seals", alg, i)
			}
			seen[nonce] = struct{}{}
		}
	}
}

func TestNewDEKCipher_RejectsBadKey(t *testing.T) {
	if _, err := NewDEKCipher([]byte("short"), AlgAESGCM); err == nil {
		t.Fatal("expected error for non-32-byte DEK")
	}
}
