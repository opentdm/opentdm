package codec

// KV is a parsed key/value pair (parser output / renderer input).
type KV struct {
	Key   string
	Value string
}

// Warning is a non-fatal parse note (e.g. a duplicate key) surfaced to the user
// in response metadata.
type Warning struct {
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

// ParseError is a fatal, structured parse failure with a stable code and the
// offending line (1-based).
type ParseError struct {
	Line    int
	Code    string
	Message string
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return e.Code + " (line " + itoa(e.Line) + "): " + e.Message
	}
	return e.Code + ": " + e.Message
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
