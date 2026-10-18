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
