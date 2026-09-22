package app

import (
	"QueueLite/internal/queue/domain"
	"context"

	"github.com/google/uuid"
)

type QueueRepo interface {
	CreateQueue(ctx context.Context, queue *domain.Queue) (*domain.Queue, error)
	UpdateQueue(ctx context.Context, queue *domain.Queue) error
	GetQueue(ctx context.Context, id uuid.UUID) (*domain.Queue, error)
	GetAllQueueByBusiness(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error)
	GetAllQueueByBusinessFilterState(ctx context.Context, businessID uuid.UUID, state domain.QueueState) ([]domain.Queue, error)
	DeleteQueue(ctx context.Context, id uuid.UUID) error
}
