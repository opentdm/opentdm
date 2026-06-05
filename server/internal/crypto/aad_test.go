package crypto

import "testing"

func TestVersionAAD_RoundTrip(t *testing.T) {
	c, _ := NewDEKCipher(mustDEK(t), AlgAESGCM)
	aad := VersionAAD("proj", "staging", "cfg", "variable")
	blob, err := c.Seal([]byte(`{"items":[]}`), aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := c.Open(blob, aad); err != nil {
		t.Fatalf("Open with same AAD: %v", err)
	}
}

func TestVersionAAD_BindsKind(t *testing.T) {
	c, _ := NewDEKCipher(mustDEK(t), AlgAESGCM)
	blob, _ := c.Seal([]byte("payload"), VersionAAD("p", "e", "c", "variable"))
	// A snapshot sealed as "variable" must not open as "file".
	if _, err := c.Open(blob, VersionAAD("p", "e", "c", "file")); err == nil {
		t.Fatal("expected Open to fail when snapshot kind differs")
	}
}

func TestVersionAAD_DistinctFromBlobAndItem(t *testing.T) {
	c, _ := NewDEKCipher(mustDEK(t), AlgAESGCM)
	// A version snapshot can't be reinterpreted as a live file blob (different tag).
	vblob, _ := c.Seal([]byte("x"), VersionAAD("p", "e", "c", "file"))
	if _, err := c.Open(vblob, BlobAAD("p", "e", "c")); err == nil {
		t.Fatal("version blob must not open as a file blob (tag binding)")
	}
	// And a file blob can't be reinterpreted as a version snapshot.
	fblob, _ := c.Seal([]byte("x"), BlobAAD("p", "e", "c"))
	if _, err := c.Open(fblob, VersionAAD("p", "e", "c", "file")); err == nil {
		t.Fatal("file blob must not open as a version snapshot")
	}
}

func TestVersionAAD_BindsEnv(t *testing.T) {
	c, _ := NewDEKCipher(mustDEK(t), AlgAESGCM)
	blob, _ := c.Seal([]byte("x"), VersionAAD("p", "staging", "c", "file"))
	if _, err := c.Open(blob, VersionAAD("p", "production", "c", "file")); err == nil {
		t.Fatal("version snapshot must be bound to its environment")
	}
	// Base ("") and a named env must not be interchangeable.
	base, _ := c.Seal([]byte("x"), VersionAAD("p", "", "c", "file"))
	if _, err := c.Open(base, VersionAAD("p", "staging", "c", "file")); err == nil {
		t.Fatal("base snapshot must not open as an env snapshot")
	}
}
