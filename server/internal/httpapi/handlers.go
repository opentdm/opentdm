package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/opentdm/opentdm/server/internal/app"
)

// Handlers holds the dependencies shared by all HTTP handlers.
type Handlers struct {
	svc           *app.Service
	logger        *slog.Logger
	secureCookies bool
}

// NewHandlers builds the handler set. secureCookies sets the Secure flag on
// auth cookies (enable behind HTTPS).
func NewHandlers(svc *app.Service, logger *slog.Logger, secureCookies bool) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{svc: svc, logger: logger, secureCookies: secureCookies}
}

// decodeJSON reads and size-limits a JSON request body into dst.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// writeErr maps an app/domain error to an RFC 9457 problem response.
func (h *Handlers) writeErr(w http.ResponseWriter, r *http.Request, err error) {
	var ve *app.ValidationError
	switch {
	case errors.Is(err, app.ErrUnauthorized):
		WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "Unauthorized", "")
	case errors.Is(err, app.ErrForbidden):
		WriteProblem(w, r, http.StatusForbidden, "forbidden", "Forbidden", "")
	case errors.Is(err, app.ErrNotFound):
		WriteProblem(w, r, http.StatusNotFound, "not_found", "Not found", "")
	case errors.Is(err, app.ErrConflict):
		WriteProblem(w, r, http.StatusConflict, "conflict", "Conflict", "")
	case errors.As(err, &ve):
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation_error", "Validation error", ve.Error())
	default:
		h.logger.Error("internal_error", slog.String("path", r.URL.Path), slog.String("err", err.Error()))
		WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal server error", "")
	}
}

// badRequest writes a 400 for malformed input (e.g. bad JSON).
func (h *Handlers) badRequest(w http.ResponseWriter, r *http.Request, detail string) {
	WriteProblem(w, r, http.StatusBadRequest, "bad_request", "Bad request", detail)
}
