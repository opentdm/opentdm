package httpapi

import (
	"net/http"
)

// POST /projects/{project}/configs/{config}/clone  {from, to, with_values}
// Clones one object's source-env layer into a target-env layer. Responds with
// counts/version only — never any values.
func (h *Handlers) handleCloneConfig(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	var req struct {
		From       string `json:"from"`
		To         string `json:"to"`
		WithValues bool   `json:"with_values"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	version, err := h.svc.CloneLayer(r.Context(), p, c, req.From, req.To, req.WithValues, actorID(r.Context()))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"from": req.From, "to": req.To, "version": version.Version}, nil)
}

// POST /projects/{project}/clone-environment  {from, to, with_values}
// Clones every non-archived object's source-env layer into a target-env layer.
// Responds with a per-config summary of names and counts — never any values.
func (h *Handlers) handleCloneEnvironment(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	var req struct {
		From       string `json:"from"`
		To         string `json:"to"`
		WithValues bool   `json:"with_values"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	sum, err := h.svc.CloneEnvironment(r.Context(), p, req.From, req.To, req.WithValues, actorID(r.Context()))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	failed := make([]map[string]string, 0, len(sum.Failed))
	for _, f := range sum.Failed {
		failed = append(failed, map[string]string{"config": f.Config, "reason": f.Reason})
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"from":      req.From,
		"to":        req.To,
		"cloned":    strOrEmpty(sum.Cloned),
		"unchanged": strOrEmpty(sum.Unchanged),
		"skipped":   strOrEmpty(sum.Skipped),
		"failed":    failed,
	}, nil)
}

// strOrEmpty ensures a nil slice serializes as [] rather than null.
func strOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
