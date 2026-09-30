package app

import (
	businessdomain "QueueLite/internal/business/domain"
	"QueueLite/internal/queue/domain"
	userdomain "QueueLite/internal/user/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type BusinessLookup interface {
	GetBusiness(ctx context.Context, id uuid.UUID) (*businessdomain.Business, error)
}

type UserLookup interface {
	GetUser(ctx context.Context, id uuid.UUID) (*userdomain.User, error)
}

type QueueQuota interface {
	CheckBusinessQueueQuota(context.Context, uuid.UUID) error
	DecreaseBusinessCapacity(context.Context, uuid.UUID) error
	IncreaseBusinessCapacity(context.Context, uuid.UUID) error
	CheckUserPrioritySlot(context.Context, uuid.UUID) error
}

type QueueRepo interface {
	CreateQueue(ctx context.Context, queue *domain.Queue) (*domain.Queue, error)
	CreateQueues(ctx context.Context, queues []domain.Queue) error
	UpdateQueue(ctx context.Context, queue *domain.Queue) error
	GetQueue(ctx context.Context, id uuid.UUID) (*domain.Queue, error)
	GetAllQueueByBusiness(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error)
	GetActiveQueuesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Queue, error)
	// GetAllQueueByBusinessFilterState(ctx context.Context, businessID uuid.UUID, state domain.QueueState) ([]domain.Queue, error)
	DeleteQueue(ctx context.Context, id uuid.UUID) error

	GetActiveQueueByUserAndBusiness(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.Queue, error)
	GetDailyQueueNumberFloor(ctx context.Context, businessID uuid.UUID, date time.Time) (int64, error)
	// GetQueueByBusinessPrivate(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error)
	GetTopQueueByBusinessPrivateForUpdate(ctx context.Context, businessID uuid.UUID) (*domain.Queue, error)
	// GetBusinessPublicQueueSummary(ctx context.Context, businessID uuid.UUID) (*domain.PublicQueueSummary, error)
}
