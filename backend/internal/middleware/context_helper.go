package middleware

import (
	queuedomain "QueueLite/internal/queue/domain"
	userdomain "QueueLite/internal/user/domain"
	"context"
)

type contextKey string

const userIdContextKey contextKey = "userId"
const queueIdContextKey contextKey = "queueId"
const optionalUserIdcontextKey contextKey = "optionalUserId"
const loadedUserContextKey contextKey = "loadedUser"
const loadedQueueContextKey contextKey = "loadedQueue"

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

func WithLoadedUser(ctx context.Context, user *userdomain.User) context.Context {
	return context.WithValue(ctx, loadedUserContextKey, user)
}

func LoadedUserFromContext(ctx context.Context) (*userdomain.User, bool) {
	user, ok := ctx.Value(loadedUserContextKey).(*userdomain.User)
	return user, ok && user != nil
}

func WithLoadedQueue(ctx context.Context, queue *queuedomain.Queue) context.Context {
	return context.WithValue(ctx, loadedQueueContextKey, queue)
}

func LoadedQueueFromContext(ctx context.Context) (*queuedomain.Queue, bool) {
	queue, ok := ctx.Value(loadedQueueContextKey).(*queuedomain.Queue)
	return queue, ok && queue != nil
}
