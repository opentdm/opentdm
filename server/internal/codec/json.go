package codec

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

const (
	maxJSONDepth  = 200
	maxJSONTokens = 5_000_000
)

// ValidateJSON checks that raw is a single, well-formed JSON document with no
// duplicate object keys and bounded depth/token-count. It does NOT enforce a
// byte size (the HTTP boundary caps that). Duplicate keys are rejected because
// Go's decoder would silently keep the last — an ambiguity for a fixture store.
func ValidateJSON(raw []byte) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return &ParseError{Code: "E_EMPTY", Message: "empty JSON document"}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	type frame struct {
		isObject  bool
		expectKey bool
		keys      map[string]bool
	}
	var stack []*frame
	tokens, topLevel := 0, 0

	valueCompleted := func() {
		if len(stack) == 0 {
			topLevel++
			return
		}
		if top := stack[len(stack)-1]; top.isObject {
			top.expectKey = true
		}
	}

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return &ParseError{Code: "E_BAD_JSON", Message: err.Error()}
		}
		if tokens++; tokens > maxJSONTokens {
			return &ParseError{Code: "E_TOO_MANY", Message: "too many JSON tokens"}
		}

		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{':
				if len(stack) >= maxJSONDepth {
					return &ParseError{Code: "E_TOO_DEEP", Message: "JSON nested too deeply"}
				}
				stack = append(stack, &frame{isObject: true, expectKey: true, keys: map[string]bool{}})
			case '[':
				if len(stack) >= maxJSONDepth {
					return &ParseError{Code: "E_TOO_DEEP", Message: "JSON nested too deeply"}
				}
				stack = append(stack, &frame{})
			case '}', ']':
				stack = stack[:len(stack)-1]
				valueCompleted()
			}
			continue
		}

		if len(stack) > 0 {
			if top := stack[len(stack)-1]; top.isObject && top.expectKey {
				key, _ := tok.(string)
				if top.keys[key] {
					return &ParseError{Code: "E_DUP_KEY", Message: "duplicate key: " + key}
				}
				top.keys[key] = true
				top.expectKey = false
				continue
			}
		}
		valueCompleted()
	}

	if len(stack) != 0 {
		return &ParseError{Code: "E_BAD_JSON", Message: "unbalanced JSON"}
	}
	if topLevel != 1 {
		return &ParseError{Code: "E_BAD_JSON", Message: "expected exactly one top-level JSON value"}
	}
	return nil
}
