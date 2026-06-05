package codec

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Output formats for resolve.
const (
	FormatJSON       = "json"
	FormatDotenv     = "dotenv"
	FormatShell      = "shell"
	FormatYAML       = "yaml"
	FormatProperties = "properties"
)

// Render serializes resolved pairs in the requested format, returning the body
// and its Content-Type. Every key is validated first (defense in depth — the
// resolver should already have rejected unsafe keys). Pairs are assumed to be
// in the caller's desired order (the resolver sorts by key).
func Render(format string, pairs []KV) (body []byte, contentType string, err error) {
	for _, p := range pairs {
		if !ValidKey(p.Key) {
			return nil, "", fmt.Errorf("codec: refusing to render unsafe key %q", p.Key)
		}
	}
	switch format {
	case "", FormatJSON:
		return renderJSON(pairs)
	case FormatDotenv:
		return []byte(renderDotenv(pairs)), "text/plain; charset=utf-8", nil
	case FormatShell:
		return []byte(renderShell(pairs)), "text/plain; charset=utf-8", nil
	case FormatYAML:
		return renderYAML(pairs)
	case FormatProperties:
		return []byte(renderProperties(pairs)), "text/plain; charset=utf-8", nil
	default:
		return nil, "", fmt.Errorf("codec: unknown format %q", format)
	}
}

func renderJSON(pairs []KV) ([]byte, string, error) {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Key] = p.Value
	}
	b, err := json.Marshal(m) // marshals with sorted keys
	if err != nil {
		return nil, "", err
	}
	return b, "application/json", nil
}

func renderYAML(pairs []KV) ([]byte, string, error) {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Key] = p.Value
	}
	b, err := yaml.Marshal(m)
	if err != nil {
		return nil, "", err
	}
	return b, "application/yaml", nil
}

// dotenvSafe matches values that need no quoting in a .env file.
var dotenvSafe = regexp.MustCompile(`^[A-Za-z0-9_./:@%+-]+$`)

func renderDotenv(pairs []KV) string {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(p.Key)
		b.WriteByte('=')
		if p.Value != "" && dotenvSafe.MatchString(p.Value) {
			b.WriteString(p.Value)
		} else if p.Value == "" {
			// KEY= (present, empty)
		} else {
			b.WriteString(quoteDotenv(p.Value))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// quoteDotenv double-quotes a value, escaping backslash, quote, and control
// chars so the line is a single, unambiguous assignment.
func quoteDotenv(v string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range v {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// renderShell emits `export KEY='value'`, single-quote escaped so it is safe to
// `eval`. The only metacharacter inside single quotes is the single quote
// itself, escaped as the classic '\” sequence.
func renderShell(pairs []KV) string {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString("export ")
		b.WriteString(p.Key)
		b.WriteString("='")
		b.WriteString(strings.ReplaceAll(p.Value, "'", `'\''`))
		b.WriteString("'\n")
	}
	return b.String()
}

// renderProperties emits Java .properties lines, escaping the separator/special
// characters and newlines.
func renderProperties(pairs []KV) string {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(escapeProperty(p.Key, true))
		b.WriteByte('=')
		b.WriteString(escapeProperty(p.Value, false))
		b.WriteByte('\n')
	}
	return b.String()
}

func escapeProperty(s string, isKey bool) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '=', ':':
			b.WriteByte('\\')
			b.WriteRune(r)
		case ' ':
			if isKey {
				b.WriteString(`\ `)
			} else {
				b.WriteByte(' ')
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
