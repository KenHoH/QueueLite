package middleware

import (
	"context"
)

type contextKey string

const userIdContextKey contextKey = "userId"
const queueIdContextKey contextKey = "queueId"
const optionalUserIdcontextKey = "optionalUserId"

func WithUserId(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, userIdContextKey, userId)
}

func UserIdFromContext(ctx context.Context) (string, bool) {
	userId, ok := ctx.Value(userIdContextKey).(string)
	return userId, ok
}

func WithOptionalUserId(ctx context.Context, userId string) context.Context {
	return context.WithValue(ctx, optionalUserIdcontextKey, userId)
}

func OptionalUserIdFromContext(ctx context.Context) (string, bool) {
	userId, ok := ctx.Value(optionalUserIdcontextKey).(string)
	return userId, ok
}

func WithQueueId(ctx context.Context, queueId string) context.Context {
	return context.WithValue(ctx, queueIdContextKey, queueId)
}

func QueueIdFromContext(ctx context.Context) (string, bool) {
	queueId, ok := ctx.Value(queueIdContextKey).(string)
	return queueId, ok
}
