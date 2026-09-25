package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/queue/app"
	"QueueLite/internal/queue/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QueueRepoImpl struct {
	db *gorm.DB
}

func NewQueueRepo(db *gorm.DB) *QueueRepoImpl {
	return &QueueRepoImpl{db: db}
}

func (r *QueueRepoImpl) CreateQueue(ctx context.Context, queue *domain.Queue) (*domain.Queue, error) {
	record := toQueueRecord(queue)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create queue: %w", err)
	}
	return toDomainQueue(&record), nil
}

func (r *QueueRepoImpl) UpdateQueue(ctx context.Context, queue *domain.Queue) error {
	result := r.db.WithContext(ctx).Model(&model.Queue{}).Where("id = ?", queue.ID).Updates(map[string]any{
		"business_id":          queue.BusinessID,
		"user_id":              queue.UserID,
		"called_by_counter_id": queue.CalledByCounterID,
		"name":                 queue.Name,
		"state":                model.QueueState(queue.State),
		"priority":             queue.Priority,
		"called_at":            queue.CalledAt,
		"processing_at":        queue.ProcessingAt,
		"done_at":              queue.DoneAt,
		"cancelled_at":         queue.CancelledAt,
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
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&record).Error
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
	if err := r.db.WithContext(ctx).Where("business_id = ?", businessID).Order("priority DESC, created_at ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("get business queues: %w", err)
	}
	return toDomainQueues(records), nil
}

func (r *QueueRepoImpl) GetAllQueueByBusinessFilterState(ctx context.Context, businessID uuid.UUID, state domain.QueueState) ([]domain.Queue, error) {
	var records []model.Queue
	if err := r.db.WithContext(ctx).Where("business_id = ? AND state = ?", businessID, model.QueueState(state)).Order("priority DESC, created_at ASC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("get business queues by state: %w", err)
	}
	return toDomainQueues(records), nil
}

func (r *QueueRepoImpl) DeleteQueue(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Queue{})
	if result.Error != nil {
		return fmt.Errorf("delete queue: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrQueueNotFound, id)
	}
	return nil
}

func (r *QueueRepoImpl) GetActiveQueueByUserAndBusiness(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.Queue, error) {
	var record model.Queue
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND user_id = ? AND state IN ?", businessID, userID, activeQueueStates()).
		Order("created_at ASC").
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active user queue: %w", err)
	}
	return toDomainQueue(&record), nil
}

func (r *QueueRepoImpl) GenerateDailyQueueName(ctx context.Context, businessID uuid.UUID, date time.Time) (string, error) {
	day := date.Truncate(24 * time.Hour)
	nextDay := day.Add(24 * time.Hour)
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Queue{}).
		Where("business_id = ? AND created_at >= ? AND created_at < ?", businessID, day, nextDay).
		Count(&count).Error; err != nil {
		return "", fmt.Errorf("generate daily queue name: %w", err)
	}
	return fmt.Sprintf("A%03d", count+1), nil
}

func (r *QueueRepoImpl) GetQueueByBusinessPrivate(ctx context.Context, businessID uuid.UUID) ([]domain.Queue, error) {
	return r.GetAllQueueByBusinessFilterState(ctx, businessID, domain.QueueStateWaiting)
}

func (r *QueueRepoImpl) GetTopQueueByBusinessPrivateForUpdate(ctx context.Context, businessID uuid.UUID) (*domain.Queue, error) {
	var record model.Queue
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("business_id = ? AND state = ?", businessID, model.QueueStateWaiting).
		Order("priority DESC, created_at ASC").
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get top business queue: %w", err)
	}
	return toDomainQueue(&record), nil
}

func (r *QueueRepoImpl) GetBusinessPublicQueueSummary(ctx context.Context, businessID uuid.UUID) (*domain.PublicQueueSummary, error) {
	var current model.Queue
	var currentName *string
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND state IN ?", businessID, []model.QueueState{model.QueueStateCalled, model.QueueStateProcessing}).
		Order("called_at ASC NULLS LAST, created_at ASC").
		Take(&current).Error
	if err == nil {
		currentName = &current.Name
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get current public queue: %w", err)
	}

	var totalWaiting int64
	if err := r.db.WithContext(ctx).Model(&model.Queue{}).Where("business_id = ? AND state = ?", businessID, model.QueueStateWaiting).Count(&totalWaiting).Error; err != nil {
		return nil, fmt.Errorf("count waiting queues: %w", err)
	}

	next, err := r.GetTopQueueByBusinessPrivateForUpdate(ctx, businessID)
	if err != nil {
		return nil, err
	}
	var nextName *string
	if next != nil {
		nextName = &next.Name
	}
	return &domain.PublicQueueSummary{CurrentQueueName: currentName, TotalWaiting: totalWaiting, NextQueueName: nextName}, nil
}

func activeQueueStates() []model.QueueState {
	return []model.QueueState{model.QueueStateWaiting, model.QueueStateCalled, model.QueueStateProcessing}
}
