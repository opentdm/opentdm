package httpapi

import (
	"io"
	"net/http"

	"github.com/opentdm/opentdm/server/internal/model"
)

const defaultMaxBlobBytes = 10 << 20

func fileContentType(format string) string {
	switch format {
	case model.FormatJSON:
		return "application/json"
	case model.FormatCSV:
		return "text/csv; charset=utf-8"
	case model.FormatXML:
		return "application/xml"
	case model.FormatYAML:
		return "application/yaml"
	default:
		return "application/octet-stream"
	}
}

// PUT /projects/{project}/configs/{config}/blob?env=&comment=
// Raw request body is the file content (size-limited). Cuts a version.
func (h *Handlers) handlePutBlob(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	limit := h.maxBlobBytes
	if limit <= 0 {
		limit = defaultMaxBlobBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	content, err := io.ReadAll(r.Body)
	if err != nil {
		WriteProblem(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "File too large", "")
		return
	}
	env := r.URL.Query().Get("env")
	version, err := h.svc.SetBlob(r.Context(), p, c, env, content, limit, strPtr(r.URL.Query().Get("comment")), actorID(r.Context()))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"env": env, "size": len(content), "version": version.Version}, nil)
}

// GET /projects/{project}/configs/{config}/blob?env=  -> decrypted file bytes.
func (h *Handlers) handleGetBlob(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	content, err := h.svc.GetBlob(r.Context(), p, c, r.URL.Query().Get("env"))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	w.Header().Set("Content-Type", fileContentType(c.Format))
	w.Header().Set("Cache-Control", "no-store, private")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
