// Package httpapi contains the Chi router, HTTP handlers, and middleware. The
// handlers are intentionally thin: decode -> validate -> call a service ->
// render. The canonical success envelope is {"data":…,"error":null,"meta":…};
// errors use RFC 9457 application/problem+json.
package httpapi

import (
	"encoding/json"
	"net/http"
)

// Envelope is the success response shape for management/JSON endpoints.
type Envelope struct {
	Data  any `json:"data"`
	Error any `json:"error"`
	Meta  any `json:"meta,omitempty"`
}

// Problem is an RFC 9457 problem+json error body.
type Problem struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Code     string `json:"code,omitempty"`
}

// WriteJSON writes a success envelope with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data, Error: nil, Meta: meta})
}

// WriteProblem writes an RFC 9457 error response.
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	p := Problem{
		Type:   "https://opentdm.dev/errors/" + code,
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
	}
	if r != nil {
		p.Instance = r.URL.Path
	}
	_ = json.NewEncoder(w).Encode(p)
}
