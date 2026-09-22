package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/counter/domain"
)

func toCounterRecord(counter *domain.Counter) model.Counter {
	return model.Counter{
		ID:                counter.ID,
		BusinessID:        counter.BusinessID,
		Name:              counter.Name,
		CurrentEmployeeID: counter.CurrentEmployeeID,
		CurrentQueueID:    counter.CurrentQueueID,
	}
}

func toDomainCounter(record *model.Counter) *domain.Counter {
	return &domain.Counter{
		ID:                record.ID,
		BusinessID:        record.BusinessID,
		Name:              record.Name,
		CurrentEmployeeID: record.CurrentEmployeeID,
		CurrentQueueID:    record.CurrentQueueID,
	}
}
