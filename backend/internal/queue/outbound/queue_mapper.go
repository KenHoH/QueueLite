package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/queue/domain"
)

func toQueueRecord(queue *domain.Queue) model.Queue {
	return model.Queue{
		ID:                queue.ID,
		BusinessID:        queue.BusinessID,
		UserID:            queue.UserID,
		CalledByCounterID: queue.CalledByCounterID,
		Name:              queue.Name,
		State:             model.QueueState(queue.State),
		Priority:          queue.Priority,
		CreatedAt:         queue.CreatedAt,
		UpdatedAt:         queue.UpdatedAt,
	}
}

func toDomainQueue(record *model.Queue) *domain.Queue {
	return &domain.Queue{
		ID:                record.ID,
		BusinessID:        record.BusinessID,
		UserID:            record.UserID,
		CalledByCounterID: record.CalledByCounterID,
		Name:              record.Name,
		State:             domain.QueueState(record.State),
		Priority:          record.Priority,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func toDomainQueues(records []model.Queue) []domain.Queue {
	queues := make([]domain.Queue, 0, len(records))
	for i := range records {
		queues = append(queues, *toDomainQueue(&records[i]))
	}
	return queues
}
