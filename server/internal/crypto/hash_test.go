package crypto

import (
	"bytes"
	"testing"
)

func TestTokenHash_DeterministicAndPeppered(t *testing.T) {
	pepper := []byte("server-pepper")
	tok := []byte("otdm_abc123")
	h1 := TokenHash(pepper, tok)
	h2 := TokenHash(pepper, tok)
	if !bytes.Equal(h1, h2) {
		t.Fatal("TokenHash must be deterministic")
	}
	if bytes.Equal(h1, TokenHash([]byte("different-pepper"), tok)) {
		t.Fatal("TokenHash must depend on the pepper")
	}
	if bytes.Equal(h1, TokenHash(pepper, []byte("otdm_other"))) {
		t.Fatal("TokenHash must depend on the token")
	}
	if len(h1) != 32 {
		t.Fatalf("expected 32-byte HMAC-SHA256, got %d", len(h1))
	}
}

func TestContentHMAC_KeyedByDEK(t *testing.T) {
	dek1 := mustDEK(t)
	dek2 := mustDEK(t)
	pt := []byte("true")
	h1, err := ContentHMAC(dek1, pt)
	if err != nil {
		t.Fatalf("ContentHMAC: %v", err)
	}
	again, _ := ContentHMAC(dek1, pt)
	if !bytes.Equal(h1, again) {
		t.Fatal("ContentHMAC must be deterministic per DEK")
	}
	// Different DEK => different hash, so it is not a global confirmation oracle.
	h2, _ := ContentHMAC(dek2, pt)
	if bytes.Equal(h1, h2) {
		t.Fatal("ContentHMAC must differ across DEKs (keyed, not a raw hash)")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	a := []byte("abcdef")
	if !ConstantTimeEqual(a, []byte("abcdef")) {
		t.Fatal("equal slices should compare equal")
	}
	if ConstantTimeEqual(a, []byte("abcdeg")) {
		t.Fatal("different slices should not compare equal")
	}
	if ConstantTimeEqual(a, []byte("abc")) {
		t.Fatal("different-length slices should not compare equal")
	}
}
