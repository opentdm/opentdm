package codec

import (
	"bytes"
	"testing"
)

func TestValidateJSON(t *testing.T) {
	ok := []string{`{"a":1,"b":[1,2,{"c":true}]}`, `[]`, `"string"`, `42`, `{"nested":{"x":{"y":1}}}`}
	for _, s := range ok {
		if err := ValidateJSON([]byte(s)); err != nil {
			t.Errorf("ValidateJSON(%s) = %v, want nil", s, err)
		}
	}
	bad := map[string]string{
		"dup key":          `{"a":1,"a":2}`,
		"two top-level":    `1 2`,
		"trailing garbage": `{} x`,
		"malformed":        `{`,
		"empty":            ``,
		"nested dup":       `{"o":{"k":1,"k":2}}`,
	}
	for name, s := range bad {
		if err := ValidateJSON([]byte(s)); err == nil {
			t.Errorf("ValidateJSON(%q) [%s] = nil, want error", s, name)
		}
	}
}

func TestValidateJSON_DepthBomb(t *testing.T) {
	var b bytes.Buffer
	for i := 0; i < maxJSONDepth+5; i++ {
		b.WriteByte('[')
	}
	for i := 0; i < maxJSONDepth+5; i++ {
		b.WriteByte(']')
	}
	if err := ValidateJSON(b.Bytes()); err == nil {
		t.Fatal("expected depth-bomb to be rejected")
	}
}

func TestValidateCSV(t *testing.T) {
	if err := ValidateCSV([]byte("a,b,c\n1,2,3\n")); err != nil {
		t.Errorf("valid CSV: %v", err)
	}
	if err := ValidateCSV([]byte("a,b\n1,2,3\n")); err != nil {
		t.Errorf("ragged CSV should be allowed: %v", err)
	}
	if err := ValidateCSV([]byte("")); err == nil {
		t.Error("empty CSV should be rejected")
	}
	if err := ValidateCSV([]byte("a,\"unterminated\nb")); err == nil {
		t.Error("malformed CSV should be rejected")
	}
}

func TestValidateXML_RejectsDOCTYPE(t *testing.T) {
	xxe := `<?xml version="1.0"?>
<!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
<foo>&xxe;</foo>`
	if err := ValidateXML([]byte(xxe)); err == nil {
		t.Fatal("XXE payload (DOCTYPE) must be rejected")
	}
	if err := ValidateXML([]byte(`<root><a>1</a></root>`)); err != nil {
		t.Errorf("valid XML: %v", err)
	}
	if err := ValidateXML([]byte(`<root><a></root>`)); err == nil {
		t.Error("malformed XML should be rejected")
	}
}

func TestValidateFile(t *testing.T) {
	if err := ValidateFile(FormatJSON, []byte(`{"a":1}`)); err != nil {
		t.Errorf("json: %v", err)
	}
	if err := ValidateFile("env", []byte("A=1")); err == nil {
		t.Error("env is not a file format; should error")
	}
}

func TestCanonicalVarSnapshot_DeterministicAndRoundTrips(t *testing.T) {
	a := []SnapshotItem{{Key: "B", Value: "2"}, {Key: "A", Value: "1", IsSecret: true}}
	b := []SnapshotItem{{Key: "A", Value: "1", IsSecret: true}, {Key: "B", Value: "2"}}
	ca, _ := CanonicalVarSnapshot(a)
	cb, _ := CanonicalVarSnapshot(b)
	if !bytes.Equal(ca, cb) {
		t.Fatalf("snapshot not order-independent:\n%s\n%s", ca, cb)
	}
	items, err := ParseVarSnapshot(ca)
	if err != nil {
		t.Fatalf("ParseVarSnapshot: %v", err)
	}
	if len(items) != 2 || items[0].Key != "A" || !items[0].IsSecret {
		t.Fatalf("round-trip mismatch: %+v", items)
	}
}

func TestCanonicalVarSnapshot_DeletedNormalizesValue(t *testing.T) {
	c1, _ := CanonicalVarSnapshot([]SnapshotItem{{Key: "X", Value: "anything", Deleted: true}})
	c2, _ := CanonicalVarSnapshot([]SnapshotItem{{Key: "X", Value: "", Deleted: true}})
	if !bytes.Equal(c1, c2) {
		t.Fatal("deleted item value must be normalized to empty for stable hashing")
	}
}
