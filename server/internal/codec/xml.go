package codec

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

const maxXMLDepth = 256

// ValidateXML checks that raw is well-formed XML and rejects DOCTYPE
// declarations entirely (kills XXE / billion-laughs / DTD ambiguity). Go's
// encoding/xml does not resolve external entities by default; we additionally
// disable custom entities and reject any DOCTYPE directive.
func ValidateXML(raw []byte) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return &ParseError{Code: "E_EMPTY", Message: "empty XML document"}
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = true
	dec.Entity = map[string]string{} // no custom entity expansion
	dec.CharsetReader = nil          // reject unknown/non-UTF-8 charsets (no external lookups)

	depth := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return &ParseError{Code: "E_BAD_XML", Message: err.Error()}
		}
		switch t := tok.(type) {
		case xml.Directive:
			if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(string(t))), "DOCTYPE") {
				return &ParseError{Code: "E_XML_DOCTYPE", Message: "DOCTYPE is not allowed"}
			}
		case xml.StartElement:
			if depth++; depth > maxXMLDepth {
				return &ParseError{Code: "E_TOO_DEEP", Message: "XML nested too deeply"}
			}
		case xml.EndElement:
			depth--
		}
	}
	return nil
}

// ValidateFile validates raw against the given file format. Unknown formats
// return an error.
func ValidateFile(format string, raw []byte) error {
	switch format {
	case FormatJSON:
		return ValidateJSON(raw)
	case FormatCSV:
		return ValidateCSV(raw)
	case FormatXML:
		return ValidateXML(raw)
	default:
		return &ParseError{Code: "E_UNSUPPORTED", Message: "not a file format: " + format}
	}
}
