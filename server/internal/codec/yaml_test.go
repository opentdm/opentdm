package codec

import "testing"

func TestValidateYAML(t *testing.T) {
	if err := ValidateYAML([]byte("a: 1\nb: two\n")); err != nil {
		t.Errorf("valid yaml rejected: %v", err)
	}
	if err := ValidateYAML([]byte("   \n")); err == nil {
		t.Error("empty yaml accepted")
	}
	if err := ValidateYAML([]byte("a:\n\tb: 1\n")); err == nil {
		t.Error("tab-indented yaml accepted")
	}
	if err := ValidateFile(FormatYAML, []byte("k: v\n")); err != nil {
		t.Errorf("ValidateFile(yaml) rejected valid doc: %v", err)
	}
}

// The render path is fail-closed on keys that aren't valid env names (the
// inject-safety guard), for every output format — so a properties bundle with
// dotted keys can never be rendered. This is why the upload UI rejects dotted
// .properties keys up front rather than relaxing the guard.
func TestRender_RejectsUnsafeKeys(t *testing.T) {
	for _, f := range []string{FormatShell, FormatDotenv, FormatJSON, FormatYAML} {
		if _, _, err := Render(f, []KV{{Key: "a.b.c", Value: "x"}}); err == nil {
			t.Errorf("Render(%s) accepted an unsafe (dotted) key", f)
		}
	}
}
