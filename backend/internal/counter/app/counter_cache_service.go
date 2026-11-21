package app

import (
	"context"

	cache "QueueLite/internal/adapter/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (s *CounterService) GetQueueNameCache(ctx context.Context, queueID string) (string, error) {
	return cache.GetQueueName(ctx, s.rdt, queueID)
}

func (s *CounterService) GetQueueCache(ctx context.Context, businessID string) ([]redis.Z, error) {
	return cache.GetWaitingQueues(ctx, businessID, s.rdt)
}

func (s *CounterService) GetWaitingQueueSnapshot(ctx context.Context, businessID uuid.UUID) ([]QueueSnapshotItem, error) {
	entries, err := cache.GetWaitingQueueSnapshot(ctx, s.rdt, businessID.String())
	if err != nil {
		return nil, err
	}
	response := make([]QueueSnapshotItem, len(entries))
	for i, entry := range entries {
		response[i] = QueueSnapshotItem{QueueID: entry.QueueID, QueueName: entry.QueueName}
	}
	return response, nil
}
