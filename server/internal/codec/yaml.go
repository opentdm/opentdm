package codec

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// ValidateYAML reports whether raw is a non-empty, parseable YAML document.
// It is the file-format validator counterpart to ValidateJSON/CSV/XML.
func ValidateYAML(raw []byte) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return &ParseError{Code: "E_EMPTY", Message: "empty YAML document"}
	}
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return &ParseError{Code: "E_BAD_YAML", Message: err.Error()}
	}
	return nil
}
