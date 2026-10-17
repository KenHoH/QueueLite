package app

import (
	"context"
	"errors"

	"QueueLite/internal/queue/domain"

	cache "QueueLite/internal/adapter/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (s *QueueService) GetQueueCache(ctx context.Context, businessID string) ([]redis.Z, error) {
	return cache.GetWaitingQueues(ctx, businessID, s.rdt)
}

func (s *QueueService) GetQueueNameCache(ctx context.Context, queueID string) (string, error) {
	return cache.GetQueueName(ctx, s.rdt, queueID)
}

func (s *QueueService) SubscribeQueueChannel(ctx context.Context, businessID string) *redis.PubSub {
	return cache.SubscribeChannel(ctx, s.rdt, cache.QueueEventChannel(businessID))
}

func (s *QueueService) publishQueueUpdate(ctx context.Context, businessID uuid.UUID) error {
	return cache.PublishMessage(ctx, s.rdt, cache.QueueEventChannel(businessID.String()), "queue.update")
}

func (s *QueueService) syncQueueCache(ctx context.Context, queue domain.Queue) error {
	if queue.State == domain.QueueStateWaiting {
		return cache.AddWaitingQueue(ctx, s.rdt, queue) // adding to queue cache
	}
	if err := cache.SyncCustomerQueue(ctx, s.rdt, queue); err != nil {
		return err
	}
	return cache.RemoveWaitingQueue(ctx, s.rdt, queue.BusinessID.String(), queue.ID.String())
}

func (s *QueueService) QueueExistsForAccess(ctx context.Context, queueID uuid.UUID) (bool, error) {
	existsInCache, err := cache.QueueExists(ctx, s.rdt, queueID.String())
	if err == nil && existsInCache {
		return true, nil
	}

	_, err = s.repo.GetQueue(ctx, queueID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrQueueNotFound) {
		return false, nil
	}
	return false, err
}

func (s *QueueService) GetWaitingQueueSnapshot(ctx context.Context, businessID uuid.UUID) ([]QueueSnapshotItem, error) {
	tasks, err := s.GetQueueCache(ctx, businessID.String())
	if err != nil {
		return nil, err
	}

	response := make([]QueueSnapshotItem, 0, len(tasks))
	for _, data := range tasks {
		queueID, ok := data.Member.(string)
		if !ok {
			continue
		}

		queueName, err := s.GetQueueNameCache(ctx, queueID)
		if err != nil {
			continue
		}

		response = append(response, QueueSnapshotItem{
			QueueID:   queueID,
			QueueName: queueName,
		})
	}
	return response, nil
}
