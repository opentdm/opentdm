package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/opentdm/opentdm/server/internal/model"
)

// GET /configs/{config}/versions?env=  -> version metadata, newest first.
func (h *Handlers) handleListVersions(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	env := r.URL.Query().Get("env")
	versions, deltas, err := h.svc.ListVersionsWithDeltas(r.Context(), p, c, env)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	out := make([]versionMetaDTO, 0, len(versions))
	for _, v := range versions {
		out = append(out, toVersionMetaDTOWithDelta(v, deltas[v.Version]))
	}
	WriteJSON(w, http.StatusOK, out, map[string]string{"env": env})
}

// GET /configs/{config}/versions/{version}?env=&reveal=  -> decrypted snapshot.
// {version} is an integer or "current".
func (h *Handlers) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	env := r.URL.Query().Get("env")
	version := 0 // "current"
	if vs := chi.URLParam(r, "version"); vs != "current" {
		n, err := strconv.Atoi(vs)
		if err != nil {
			h.badRequest(w, r, "version must be an integer or 'current'")
			return
		}
		version = n
	}
	reveal := r.URL.Query().Get("reveal") == "true"
	snap, err := h.svc.GetVersion(r.Context(), p, c, env, version, reveal)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	if snap.Kind == model.KindFile {
		w.Header().Set("Content-Type", fileContentType(c.Format))
		w.Header().Set("Cache-Control", "no-store, private")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(snap.File)
		return
	}
	items := make([]itemDTO, 0, len(snap.Vars))
	for _, v := range snap.Vars {
		items = append(items, itemDTO{Key: v.Key, Value: v.Value, IsSecret: v.IsSecret})
	}
	WriteJSON(w, http.StatusOK, map[string]any{"version": snap.Version, "kind": snap.Kind, "items": items}, nil)
}

// GET /configs/{config}/diff?env=&from=&to=  -> structured/text diff.
func (h *Handlers) handleDiff(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	env := r.URL.Query().Get("env")
	from := atoiDefault(r.URL.Query().Get("from"), 0) // 0 = empty snapshot
	to := atoiDefault(r.URL.Query().Get("to"), 0)     // 0 = current
	res, err := h.svc.Diff(r.Context(), p, c, env, from, to)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toDiffDTO(res), nil)
}

// POST /configs/{config}/rollback  {env, to_version, comment}
func (h *Handlers) handleRollback(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProjectRole(w, r, roleEditor)
	if !ok {
		return
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
		return
	}
	var req struct {
		Env       string `json:"env"`
		ToVersion int    `json:"to_version"`
		Comment   string `json:"comment"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		h.badRequest(w, r, "invalid JSON body")
		return
	}
	version, err := h.svc.Rollback(r.Context(), p, c, req.Env, req.ToVersion, strPtr(req.Comment), actorID(r.Context()))
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"env": req.Env, "version": version.Version}, nil)
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
