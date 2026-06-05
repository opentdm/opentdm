// Package codec parses and renders opentdm's variable formats (.env,
// properties) and validates file formats (json/csv/xml — Phase 2). The
// renderers are the security-sensitive output path: shell/dotenv values are
// escaped so a stored value can never become code on injection.
package codec

import (
	"regexp"
	"strings"
)

var keyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidKey reports whether key is a safe environment-variable name. It enforces
// the POSIX-ish shape and rejects the Shellshock `BASH_FUNC_*` smuggling prefix.
// The regex already forbids '=', whitespace, NUL, and newlines.
func ValidKey(key string) bool {
	if !keyRe.MatchString(key) {
		return false
	}
	if strings.HasPrefix(key, "BASH_FUNC_") {
		return false
	}
	return true
}
