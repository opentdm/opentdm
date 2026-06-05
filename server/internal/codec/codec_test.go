package codec

import (
	"strings"
	"testing"
)

func TestValidKey(t *testing.T) {
	valid := []string{"PORT", "DATABASE_URL", "_X", "a1_b2"}
	for _, k := range valid {
		if !ValidKey(k) {
			t.Errorf("expected %q valid", k)
		}
	}
	invalid := []string{"", "1ABC", "A-B", "A.B", "A B", "A=B", "A\nB", "BASH_FUNC_x", "FOO="}
	for _, k := range invalid {
		if ValidKey(k) {
			t.Errorf("expected %q invalid", k)
		}
	}
}

func TestParseDotenv_Basic(t *testing.T) {
	kvs, warns, err := ParseDotenv([]byte("# comment\nexport PORT=3000\nLOG=\"deb\\nug\"\nQ='lit$(x)'\nEMPTY=\n"))
	if err != nil {
		t.Fatalf("ParseDotenv: %v", err)
	}
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	got := map[string]string{}
	for _, kv := range kvs {
		got[kv.Key] = kv.Value
	}
	if got["PORT"] != "3000" {
		t.Errorf("PORT=%q", got["PORT"])
	}
	if got["LOG"] != "deb\nug" {
		t.Errorf("LOG=%q (want escape processed)", got["LOG"])
	}
	if got["Q"] != "lit$(x)" {
		t.Errorf("Q=%q (single quotes literal)", got["Q"])
	}
	if v, ok := got["EMPTY"]; !ok || v != "" {
		t.Errorf("EMPTY should be present and empty, got %q ok=%v", v, ok)
	}
}

func TestParseDotenv_DuplicateWarns(t *testing.T) {
	kvs, warns, err := ParseDotenv([]byte("A=1\nA=2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(kvs) != 1 || kvs[0].Value != "2" {
		t.Fatalf("dup should keep last: %+v", kvs)
	}
	if len(warns) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warns))
	}
}

func TestParseDotenv_Errors(t *testing.T) {
	if _, _, err := ParseDotenv([]byte("NOEQUALS\n")); err == nil {
		t.Error("expected error for line without '='")
	}
	if _, _, err := ParseDotenv([]byte("1BAD=x\n")); err == nil {
		t.Error("expected error for invalid key")
	}
}

func TestParseDotenv_CRLFandBOM(t *testing.T) {
	kvs, _, err := ParseDotenv([]byte("\ufeffA=1\r\nB=2\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(kvs) != 2 || kvs[0].Key != "A" || kvs[1].Key != "B" {
		t.Fatalf("BOM/CRLF handling failed: %+v", kvs)
	}
}

func TestParseProperties(t *testing.T) {
	kvs, _, err := ParseProperties([]byte("# c\n! c2\na.b = 1\nc:2\nd 3\ncont = x\\\ny\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, kv := range kvs {
		got[kv.Key] = kv.Value
	}
	if got["a.b"] != "1" || got["c"] != "2" || got["d"] != "3" {
		t.Fatalf("separator handling: %+v", got)
	}
	if got["cont"] != "xy" {
		t.Fatalf("line continuation: %q", got["cont"])
	}
}

func TestRender_RejectsUnsafeKey(t *testing.T) {
	if _, _, err := Render(FormatDotenv, []KV{{Key: "BASH_FUNC_x", Value: "1"}}); err == nil {
		t.Fatal("expected render to reject unsafe key")
	}
}

func TestRenderShell_InjectionSafe(t *testing.T) {
	// A value attempting command injection must be inertly single-quoted.
	body, ct, err := Render(FormatShell, []KV{{Key: "X", Value: `'; rm -rf / #`}})
	if err != nil {
		t.Fatal(err)
	}
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("content-type %q", ct)
	}
	want := `export X=''\''; rm -rf / #'` + "\n"
	if string(body) != want {
		t.Fatalf("shell escaping wrong:\n got %q\nwant %q", body, want)
	}
}

func TestRenderShell_CommandSubstitutionLiteral(t *testing.T) {
	body, _, _ := Render(FormatShell, []KV{{Key: "X", Value: "$(whoami)"}})
	if string(body) != "export X='$(whoami)'\n" {
		t.Fatalf("got %q", body)
	}
}

func TestRenderDotenv_QuotingAndEmpty(t *testing.T) {
	body, _, err := Render(FormatDotenv, []KV{
		{Key: "SIMPLE", Value: "abc123"},
		{Key: "SPACED", Value: "a b"},
		{Key: "NEWLINE", Value: "line1\nline2"},
		{Key: "EMPTY", Value: ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	for _, want := range []string{
		"SIMPLE=abc123\n",
		"SPACED=\"a b\"\n",
		"NEWLINE=\"line1\\nline2\"\n",
		"EMPTY=\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("dotenv output missing %q\nfull:\n%s", want, s)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	body, ct, err := Render(FormatJSON, []KV{{Key: "B", Value: "2"}, {Key: "A", Value: "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if ct != "application/json" {
		t.Errorf("ct=%q", ct)
	}
	// json.Marshal of a map sorts keys.
	if string(body) != `{"A":"1","B":"2"}` {
		t.Fatalf("json=%s", body)
	}
}

func TestRenderYAMLAndProperties(t *testing.T) {
	if _, _, err := Render(FormatYAML, []KV{{Key: "A", Value: "1"}}); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	// Render outputs resolved env-var keys (ValidKey-checked); the value's '='
	// must be escaped in properties syntax.
	body, _, err := Render(FormatProperties, []KV{{Key: "A_B", Value: "x=y"}})
	if err != nil {
		t.Fatalf("properties: %v", err)
	}
	if string(body) != "A_B=x\\=y\n" {
		t.Fatalf("properties=%q", body)
	}
}

func TestRender_UnknownFormat(t *testing.T) {
	if _, _, err := Render("toml", []KV{{Key: "A", Value: "1"}}); err == nil {
		t.Fatal("expected error for unknown format")
	}
}
