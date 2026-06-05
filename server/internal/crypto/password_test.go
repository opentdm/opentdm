package crypto

import (
	"strings"
	"testing"
)

func TestPassword_HashVerify(t *testing.T) {
	enc, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(enc, "$argon2id$v=19$") {
		t.Fatalf("unexpected encoding: %s", enc)
	}
	ok, err := VerifyPassword(enc, "correct horse battery staple")
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("correct password should verify")
	}
	bad, err := VerifyPassword(enc, "wrong password")
	if err != nil {
		t.Fatalf("VerifyPassword(wrong): %v", err)
	}
	if bad {
		t.Fatal("wrong password must not verify")
	}
}

func TestPassword_DistinctSalts(t *testing.T) {
	a, _ := HashPassword("same")
	b, _ := HashPassword("same")
	if a == b {
		t.Fatal("identical passwords must produce different hashes (random salt)")
	}
}

func TestPassword_RejectsMalformed(t *testing.T) {
	if _, err := VerifyPassword("not-a-hash", "x"); err == nil {
		t.Fatal("expected error on malformed encoding")
	}
}
