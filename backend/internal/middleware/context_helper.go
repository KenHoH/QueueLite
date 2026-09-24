package middleware

import (
	"context"
)

type contextKey string

const userIdContextKey contextKey = "userId"

func WithUserId(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userIdContextKey, userId)
}

func UserIdFromContext(ctx context.Context) (string, bool) {
	userId, ok := ctx.Value(userIdContextKey).(string)
	return userId, ok
}
