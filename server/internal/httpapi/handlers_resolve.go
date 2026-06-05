package httpapi

import (
	"net/http"
	"strconv"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/codec"
)

// GET /api/v1/projects/{project}/resolve?env=&format=&include_secrets=
//
// The consumption primitive. Accepts a session (UI) or a project+environment
// scoped service token (CI/CLI/SDK). Returns the merged variable set as a bare
// body in the requested format.
func (h *Handlers) handleResolve(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	tok, hasTok := tokenFrom(ctx)
	_, hasUser := userFrom(ctx)
	if !hasTok && !hasUser {
		WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", "")
		return
	}

	envSlug := r.URL.Query().Get("env")
	if envSlug == "" {
		h.badRequest(w, r, "missing required query parameter: env")
		return
	}
	env, err := h.svc.GetEnvironment(ctx, p.ID, envSlug)
	if err != nil {
		if err == app.ErrNotFound {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "Environment not found", "")
			return
		}
		h.writeErr(w, r, err)
		return
	}

	// Token scope enforcement (default-deny). A token may only read its own
	// project and explicitly-scoped environments.
	if hasTok {
		if tok.ProjectID != p.ID || !app.TokenAllowsEnv(tok, env.ID) {
			WriteProblem(w, r, http.StatusForbidden, "scope_denied", "Token not scoped to this project/environment", "")
			return
		}
	}

	result, err := h.svc.Resolve(ctx, p, env.ID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}

	includeSecrets := r.URL.Query().Get("include_secrets") != "false"
	pairs := make([]codec.KV, 0, len(result.Variables))
	for _, v := range result.Variables {
		if v.IsSecret && !includeSecrets {
			continue
		}
		pairs = append(pairs, codec.KV{Key: v.Key, Value: v.Value})
	}

	format := r.URL.Query().Get("format")
	body, contentType, err := codec.Render(format, pairs)
	if err != nil {
		h.badRequest(w, r, err.Error())
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("X-OpenTDM-Collisions", strconv.Itoa(len(result.Collisions)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
