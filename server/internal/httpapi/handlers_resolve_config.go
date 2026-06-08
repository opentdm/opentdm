package httpapi

import (
	"net/http"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/codec"
)

// GET /api/v1/projects/{project}/configs/{config}/resolve?env=&format=&include_secrets=
//
// The per-file consumption primitive: resolves a SINGLE variable config
// (base → env override, tombstones) and renders it in the requested format.
// Accepts a session (UI) or a project+environment scoped service token
// (CI/CLI/SDK). Unlike the project-level /resolve there is no cross-config merge,
// so there are never any collisions to report.
func (h *Handlers) handleResolveConfig(w http.ResponseWriter, r *http.Request) {
	// Authenticate before any project lookup so an anonymous caller gets a uniform
	// 401 rather than learning whether a project exists.
	ctx := r.Context()
	tok, hasTok := tokenFrom(ctx)
	_, hasUser := userFrom(ctx)
	if !hasTok && !hasUser {
		WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Authentication required", "")
		return
	}
	p, ok := h.loadProject(w, r)
	if !ok {
		return
	}
	// A session/PAT user (no service token) must be a member of the project;
	// a service token carries its own project+env scope (checked below).
	if !hasTok {
		user, _ := userFrom(ctx)
		if _, member, err := h.svc.ProjectRole(ctx, user, p.ID); err != nil {
			h.writeErr(w, r, err)
			return
		} else if !member {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "Project not found", "")
			return
		}
	}
	c, ok := h.resolveConfig(w, r, p)
	if !ok {
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

	// Token scope enforcement (default-deny): a token may only read its own
	// project and explicitly-scoped environments.
	if hasTok {
		if tok.ProjectID != p.ID || !app.TokenAllowsEnv(tok, env.ID) {
			WriteProblem(w, r, http.StatusForbidden, "scope_denied", "Token not scoped to this project/environment", "")
			return
		}
	}

	result, err := h.svc.ResolveConfig(ctx, p, c, env.ID)
	if err != nil {
		h.writeErr(w, r, err) // a file config yields a 422 validation error
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
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
