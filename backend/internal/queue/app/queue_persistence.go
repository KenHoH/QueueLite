package app

import (
	cache "QueueLite/internal/adapter/redis"
	"QueueLite/internal/apperror"
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Mutations wait for the worker so deleting a pending ticket cannot resurrect it.
func (s *QueueService) requirePersistedQueue(ctx context.Context, id uuid.UUID) error {
	_, err := s.repo.GetQueue(ctx, id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrQueueNotFound) {
		return apperror.Wrap(apperror.KindInternal, "GET_QUEUE_ERROR", "failed to get queue", err)
	}
	if s.rdt != nil {
		_, cacheErr := cache.GetCustomerQueue(ctx, s.rdt, id.String())
		if cacheErr == nil {
			return apperror.New(apperror.KindConflict, "QUEUE_PERSISTENCE_PENDING", "queue is still being saved; try again shortly")
		}
		if !errors.Is(cacheErr, redis.Nil) {
			return apperror.Wrap(apperror.KindInternal, "GET_QUEUE_CACHE_ERROR", "failed to get queue", cacheErr)
		}
	}
	return apperror.Wrap(apperror.KindNotFound, "QUEUE_NOT_FOUND", "queue not found", err)
}
