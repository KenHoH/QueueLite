package app

import (
	"QueueLite/internal/counter/domain"
	queuedomain "QueueLite/internal/queue/domain"
	"context"

	"github.com/google/uuid"
)

type CounterRepo interface {
	CreateCounter(ctx context.Context, counter *domain.Counter) (*domain.Counter, error)
	UpdateCounter(ctx context.Context, counter *domain.Counter) error
	UpdateCounterEmployee(ctx context.Context, counterID uuid.UUID, employeeID *uuid.UUID) error
	UpdateCounterCustomer(ctx context.Context, counterID uuid.UUID, queueID *uuid.UUID) error
	GetCounter(ctx context.Context, id uuid.UUID) (*domain.Counter, error)
	DeleteCounter(ctx context.Context, id uuid.UUID) error
}

type CounterQueueRepo interface {
	GetQueue(ctx context.Context, id uuid.UUID) (*queuedomain.Queue, error)
	UpdateQueue(ctx context.Context, queue *queuedomain.Queue) error
	GetTopQueueByBusinessPrivateForUpdate(ctx context.Context, businessID uuid.UUID) (*queuedomain.Queue, error)
}
