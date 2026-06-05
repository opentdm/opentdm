package crypto

import (
	"bytes"
	"testing"
)

func key(t *testing.T) []byte {
	t.Helper()
	k, err := RandomBytes(32)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestEnvKeyProvider_WrapUnwrap(t *testing.T) {
	p, err := NewEnvKeyProvider("env:v1", key(t), nil)
	if err != nil {
		t.Fatalf("NewEnvKeyProvider: %v", err)
	}
	dek := mustDEK(t)
	wrapped, ref, err := p.Wrap(dek)
	if err != nil {
		t.Fatalf("Wrap: %v", err)
	}
	if ref != "env:v1" {
		t.Fatalf("keyRef = %q, want env:v1", ref)
	}
	got, err := p.Unwrap(wrapped, ref)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("unwrapped DEK does not match original")
	}
}

func TestEnvKeyProvider_UnknownRefFails(t *testing.T) {
	p, _ := NewEnvKeyProvider("env:v1", key(t), nil)
	wrapped, _, _ := p.Wrap(mustDEK(t))
	if _, err := p.Unwrap(wrapped, "env:v2"); err == nil {
		t.Fatal("expected unwrap with unknown keyRef to fail")
	}
}

// Master key rotation: a DEK wrapped under the old key must still unwrap after
// the active key changes, as long as the old key is retained by ref.
func TestEnvKeyProvider_Rotation(t *testing.T) {
	oldKey := key(t)
	old, _ := NewEnvKeyProvider("env:v1", oldKey, nil)
	dek := mustDEK(t)
	wrapped, ref, _ := old.Wrap(dek)

	// New provider: active is v2, but it retains v1 for unwraps.
	rotated, err := NewEnvKeyProvider("env:v2", key(t), map[string][]byte{"env:v1": oldKey})
	if err != nil {
		t.Fatalf("rotated provider: %v", err)
	}
	got, err := rotated.Unwrap(wrapped, ref)
	if err != nil {
		t.Fatalf("Unwrap under retained old key: %v", err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("rotation lost the DEK")
	}
	// Re-wrap under the new active key.
	rewrapped, newRef, _ := rotated.Wrap(got)
	if newRef != "env:v2" {
		t.Fatalf("re-wrap ref = %q, want env:v2", newRef)
	}
	got2, _ := rotated.Unwrap(rewrapped, newRef)
	if !bytes.Equal(got2, dek) {
		t.Fatal("re-wrapped DEK mismatch")
	}
}

func TestEnvKeyProvider_RejectsBadKeyLength(t *testing.T) {
	if _, err := NewEnvKeyProvider("env:v1", []byte("too short"), nil); err == nil {
		t.Fatal("expected error for non-32-byte master key")
	}
}

// keyRef is bound as wrap-AAD, so tampering with stored bytes won't let an
// attacker pass it off under a different ref. Here we confirm the wrapped blob
// from one ref cannot be opened claiming another ref even if that key exists.
func TestEnvKeyProvider_RefBoundAsAAD(t *testing.T) {
	k1, k2 := key(t), key(t)
	p, _ := NewEnvKeyProvider("env:v2", k2, map[string][]byte{"env:v1": k1})
	dek := mustDEK(t)
	wrapped, _, _ := p.Wrap(dek) // wrapped under active env:v2
	// Claiming it was wrapped under env:v1 must fail authentication.
	if _, err := p.Unwrap(wrapped, "env:v1"); err == nil {
		t.Fatal("expected unwrap with mismatched keyRef to fail (AAD binding)")
	}
}
