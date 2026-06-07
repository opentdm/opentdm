package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

const auditPageDefault = 50
const auditPageMax = 200

// GET /projects/{project}/audit?limit=&before=  (viewer+) — project activity.
func (h *Handlers) handleProjectAudit(w http.ResponseWriter, r *http.Request) {
	p, ok := h.resolveProject(w, r)
	if !ok {
		return
	}
	limit := auditLimit(r)
	beforeTS, beforeID, err := parseAuditCursor(r.URL.Query().Get("before"))
	if err != nil {
		h.badRequest(w, r, "invalid cursor")
		return
	}
	entries, err := h.svc.ListProjectAudit(r.Context(), p.ID, limit+1, beforeTS, beforeID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeAuditPage(w, entries, limit)
}

// GET /audit?limit=&before=  (admin) — instance-wide activity.
func (h *Handlers) handleGlobalAudit(w http.ResponseWriter, r *http.Request) {
	limit := auditLimit(r)
	beforeTS, beforeID, err := parseAuditCursor(r.URL.Query().Get("before"))
	if err != nil {
		h.badRequest(w, r, "invalid cursor")
		return
	}
	entries, err := h.svc.ListAudit(r.Context(), limit+1, beforeTS, beforeID)
	if err != nil {
		h.writeErr(w, r, err)
		return
	}
	writeAuditPage(w, entries, limit)
}

// writeAuditPage trims the over-fetched (+1) row to detect whether a next page
// exists, so the cursor is only returned when there is genuinely more.
func writeAuditPage(w http.ResponseWriter, entries []model.AuditEntry, limit int) {
	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}
	out := make([]auditEntryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toAuditEntryDTO(e))
	}
	var meta map[string]string
	if hasMore && len(entries) > 0 {
		meta = map[string]string{"next": encodeAuditCursor(entries[len(entries)-1])}
	}
	WriteJSON(w, http.StatusOK, out, meta)
}

func auditLimit(r *http.Request) int {
	n := atoiDefault(r.URL.Query().Get("limit"), auditPageDefault)
	if n <= 0 {
		n = auditPageDefault
	}
	if n > auditPageMax {
		n = auditPageMax
	}
	return n
}

func encodeAuditCursor(e model.AuditEntry) string {
	return e.CreatedAt.UTC().Format(time.RFC3339Nano) + "_" + e.ID.String()
}

func parseAuditCursor(s string) (*time.Time, *uuid.UUID, error) {
	if s == "" {
		return nil, nil, nil
	}
	i := strings.LastIndex(s, "_")
	if i < 0 {
		return nil, nil, errors.New("bad cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, s[:i])
	if err != nil {
		return nil, nil, err
	}
	id, err := uuid.Parse(s[i+1:])
	if err != nil {
		return nil, nil, err
	}
	return &ts, &id, nil
}
