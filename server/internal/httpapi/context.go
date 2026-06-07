package httpapi

import (
	"context"

	"github.com/google/uuid"

	"github.com/opentdm/opentdm/server/internal/model"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxToken
	ctxPAT
	ctxAudit
)

// auditInfo is a per-request holder the audit middleware seeds into the context
// before the handler runs; loadProject and create-handlers fill it in so the
// middleware can record the project and target after the response.
type auditInfo struct {
	ProjectID *uuid.UUID
	TargetID  string
}

func withAuditInfo(ctx context.Context, info *auditInfo) context.Context {
	return context.WithValue(ctx, ctxAudit, info)
}

// auditInfoFrom returns the request's audit holder, or nil for non-audited
// (e.g. GET) requests where the middleware didn't seed one.
func auditInfoFrom(ctx context.Context) *auditInfo {
	v, _ := ctx.Value(ctxAudit).(*auditInfo)
	return v
}

func withUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, ctxUser, u)
}

// userFrom returns the authenticated session user, if any.
func userFrom(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(ctxUser).(model.User)
	return u, ok
}

func withToken(ctx context.Context, t model.Token) context.Context {
	return context.WithValue(ctx, ctxToken, t)
}

// tokenFrom returns the authenticated service token, if any.
func tokenFrom(ctx context.Context) (model.Token, bool) {
	t, ok := ctx.Value(ctxToken).(model.Token)
	return t, ok
}

// withPATMarker records that the request user was authenticated via a user PAT
// (a Bearer credential), so CSRF can be exempted.
func withPATMarker(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxPAT, true)
}

// isPATAuth reports whether the request was authenticated via a user PAT.
func isPATAuth(ctx context.Context) bool {
	v, _ := ctx.Value(ctxPAT).(bool)
	return v
}
