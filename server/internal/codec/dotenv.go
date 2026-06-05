package codec

import "strings"

// ParseDotenv parses a .env document into ordered key/value pairs. It accepts
// `KEY=value`, `export KEY=value`, double- and single-quoted values, `#`
// comments, blank lines, CRLF endings, and a leading UTF-8 BOM. Keys are
// validated; an invalid key or a line without `=` is a fatal ParseError.
// Duplicate keys keep the last value and emit a Warning.
func ParseDotenv(raw []byte) ([]KV, []Warning, error) {
	s := strings.TrimPrefix(string(raw), "\ufeff") // strip BOM
	lines := strings.Split(s, "\n")

	var (
		out      []KV
		warnings []Warning
		index    = map[string]int{} // key -> position in out (for dedupe)
	)

	for i, line := range lines {
		lineNo := i + 1
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, "export ")
		trimmed = strings.TrimLeft(trimmed, " \t")

		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			return nil, nil, &ParseError{Line: lineNo, Code: "E_BAD_LINE", Message: "missing '=' in assignment"}
		}
		key := strings.TrimRight(trimmed[:eq], " \t")
		if !ValidKey(key) {
			return nil, nil, &ParseError{Line: lineNo, Code: "E_BAD_KEY", Message: "invalid variable name: " + key}
		}
		value := parseDotenvValue(strings.TrimLeft(trimmed[eq+1:], " \t"))

		if pos, dup := index[key]; dup {
			warnings = append(warnings, Warning{Line: lineNo, Message: "duplicate key " + key + ": last value wins"})
			out[pos].Value = value
		} else {
			index[key] = len(out)
			out = append(out, KV{Key: key, Value: value})
		}
	}
	return out, warnings, nil
}

// parseDotenvValue handles quoting. Double quotes process escapes; single
// quotes are literal; bare values are taken verbatim (trailing whitespace
// trimmed). Inline comments are NOT stripped from bare values to avoid silent
// data loss.
func parseDotenvValue(v string) string {
	if len(v) >= 2 && v[0] == '"' {
		if end := strings.LastIndexByte(v, '"'); end > 0 {
			return unescapeDouble(v[1:end])
		}
	}
	if len(v) >= 2 && v[0] == '\'' {
		if end := strings.LastIndexByte(v, '\''); end > 0 {
			return v[1:end]
		}
	}
	return strings.TrimRight(v, " \t")
}

func unescapeDouble(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte(s[i+1])
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
