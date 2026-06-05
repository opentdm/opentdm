package codec

import "strings"

// ParseProperties parses a Java-style .properties document. It supports `#` and
// `!` comments, `=`/`:`/whitespace separators, and backslash line continuation.
// Property keys allow '.' and '-', so they are NOT subject to ValidKey (which
// governs environment-variable names); callers that intend to inject properties
// as env vars must validate separately. Duplicate keys keep the last value.
func ParseProperties(raw []byte) ([]KV, []Warning, error) {
	text := strings.TrimPrefix(string(raw), "\ufeff")
	rawLines := strings.Split(text, "\n")

	// Join backslash-continued logical lines.
	var logical []string
	var buf strings.Builder
	for _, l := range rawLines {
		l = strings.TrimRight(l, "\r")
		if buf.Len() == 0 {
			trimmed := strings.TrimLeft(l, " \t")
			if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "!") {
				continue
			}
		}
		if strings.HasSuffix(l, "\\") && !strings.HasSuffix(l, "\\\\") {
			buf.WriteString(l[:len(l)-1])
			continue
		}
		buf.WriteString(l)
		logical = append(logical, buf.String())
		buf.Reset()
	}
	if buf.Len() > 0 {
		logical = append(logical, buf.String())
	}

	var out []KV
	var warnings []Warning
	index := map[string]int{}
	for i, l := range logical {
		key, value := splitProperty(l)
		if key == "" {
			continue
		}
		if pos, dup := index[key]; dup {
			warnings = append(warnings, Warning{Line: i + 1, Message: "duplicate key " + key + ": last value wins"})
			out[pos].Value = value
		} else {
			index[key] = len(out)
			out = append(out, KV{Key: key, Value: value})
		}
	}
	return out, warnings, nil
}

// splitProperty splits a logical line at the first unescaped '=', ':', or run of
// whitespace, trimming leading whitespace of the key and surrounding space of
// the separator.
func splitProperty(line string) (string, string) {
	line = strings.TrimLeft(line, " \t\f")
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '\\' {
			i++ // skip escaped char
			continue
		}
		if c == '=' || c == ':' {
			return strings.TrimRight(line[:i], " \t\f"), strings.TrimLeft(line[i+1:], " \t\f")
		}
		if c == ' ' || c == '\t' || c == '\f' {
			return strings.TrimRight(line[:i], " \t\f"), strings.TrimLeft(line[i+1:], " \t\f=:")
		}
	}
	return strings.TrimRight(line, " \t\f"), ""
}
