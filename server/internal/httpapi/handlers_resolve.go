package httpapi

import (
	"net/http"
	"strconv"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/codec"
)

// collisionDTO is the JSON shape of a cross-config key collision in meta mode.
type collisionDTO struct {
	Key           string `json:"key"`
	WinningConfig string `json:"winning_config"`
	LosingConfig  string `json:"losing_config"`
}

// GET /api/v1/projects/{project}/resolve?env=&format=&include_secrets=&meta=
//
// The consumption primitive. Accepts a session (UI) or a project+environment
// scoped service token (CI/CLI/SDK). Returns the merged variable set as a bare
// body in the requested format.
func (h *Handlers) handleResolve(w http.ResponseWriter, r *http.Request) {
	// Authenticate before any project lookup so an anonymous caller gets a uniform
	// 401 rather than learning whether a project exists (404 vs 401).
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

	// meta=true returns the canonical JSON envelope with full cross-config
	// collision detail in meta.collisions (DECISIONS.md). The raw rendered path
	// below is unchanged (CI consumers depend on it byte-for-byte); meta mode
	// returns a JSON object and ignores the format parameter.
	if r.URL.Query().Get("meta") == "true" {
		data := make(map[string]string, len(result.Variables))
		for _, v := range result.Variables {
			if v.IsSecret && !includeSecrets {
				continue
			}
			data[v.Key] = v.Value
		}
		// Collision detail is key + config NAMES only — never values — so it is
		// safe to surface to any caller already authorized for this project+env.
		collisions := make([]collisionDTO, 0, len(result.Collisions))
		for _, c := range result.Collisions {
			collisions = append(collisions, collisionDTO{Key: c.Key, WinningConfig: c.WinningConfig, LosingConfig: c.LosingConfig})
		}
		w.Header().Set("Cache-Control", "no-store, private")
		w.Header().Set("X-OpenTDM-Collisions", strconv.Itoa(len(result.Collisions)))
		WriteJSON(w, http.StatusOK, data, map[string]any{"collisions": collisions})
		return
	}

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
