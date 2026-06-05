package httpapi

import (
	"context"

	"github.com/opentdm/opentdm/server/internal/model"
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxToken
)

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
