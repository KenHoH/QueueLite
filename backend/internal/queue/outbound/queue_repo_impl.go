package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QueueRepoImpl struct {
	db *gorm.DB
}

func NewQueueRepo(db *gorm.DB) *QueueRepoImpl {
	return &QueueRepoImpl{
		db: db,
	}
}

func (r *QueueRepoImpl) CreateQueue(ctx context.Context, queue *domain.Queue) (*domain.Queue, error) {
	record := toQueueRecord(queue)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create queue: %w", err)
	}

	return toDomainQueue(&record), nil
}

func (r *QueueRepoImpl) UpdateQueue(ctx context.Context, queue *domain.Queue) error {
	result := r.db.
		WithContext(ctx).
		Model(&model.Queue{}).
		Where("id = ?", queue.ID).
		Updates(map[string]any{
			"business_id": queue.BusinessID,
			"user_id":     queue.UserID,
			"counter_id":  queue.CounterID,
			"name":        queue.Name,
			"state":       model.QueueState(queue.State),
			"priority":    queue.Priority,
			"start_time":  queue.StartTime,
			"end_time":    queue.EndTime,
		})

	if result.Error != nil {
		return fmt.Errorf("update queue: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrQueueNotFound, queue.ID)
	}

	return nil
}

func (r *QueueRepoImpl) GetQueue(ctx context.Context, id uuid.UUID) (*domain.Queue, error) {
	var record model.Queue
	err := r.db.
		WithContext(ctx).
		Where("id = ?", id).
		Take(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrQueueNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get queue: %w", err)
	}

	return toDomainQueue(&record), nil
}

func (r *QueueRepoImpl) GetAllQueueByBusiness(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error) {
	var records []model.Queue
	if err := r.db.
		WithContext(ctx).
		Where("business_id = ?", businessID).
		Order("priority DESC, created_at ASC").
		Find(&records).
		Error; err != nil {
		return nil, fmt.Errorf("get business queues: %w", err)
	}

	return toDomainQueues(records), nil
}

func (r *QueueRepoImpl) GetAllQueueByBusinessFilterState(ctx context.Context, businessID uuid.UUID, state domain.QueueState) ([]domain.Queue, error) {
	var records []model.Queue
	if err := r.db.
		WithContext(ctx).
		Where("business_id = ? AND state = ?", businessID, model.QueueState(state)).
		Order("priority DESC, created_at ASC").
		Find(&records).
		Error; err != nil {
		return nil, fmt.Errorf("get business queues by state: %w", err)
	}

	return toDomainQueues(records), nil
}

func (r *QueueRepoImpl) DeleteQueue(ctx context.Context, id uuid.UUID) error {
	result := r.db.
		WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.Queue{})

	if result.Error != nil {
		return fmt.Errorf("delete queue: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrQueueNotFound, id)
	}

	return nil
}
