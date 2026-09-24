package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/counter/app"
	"QueueLite/internal/counter/domain"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CounterRepoImpl struct {
	db *gorm.DB
}

func NewCounterRepo(db *gorm.DB) *CounterRepoImpl {
	return &CounterRepoImpl{
		db: db,
	}
}

func (r *CounterRepoImpl) CreateCounter(ctx context.Context, counter *domain.Counter) (*domain.Counter, error) {
	record := toCounterRecord(counter)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create counter: %w", err)
	}

	return toDomainCounter(&record), nil
}

func (r *CounterRepoImpl) UpdateCounter(ctx context.Context, counter *domain.Counter) error {
	result := r.db.
		WithContext(ctx).
		Model(&model.Counter{}).
		Where("id = ?", counter.ID).
		Updates(map[string]any{
			"name": counter.Name,
		})

	if result.Error != nil {
		return fmt.Errorf("update counter: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrCounterNotFound, counter.ID)
	}

	return nil
}

func (r *CounterRepoImpl) UpdateCounterEmployee(ctx context.Context, counterID uuid.UUID, employeeID *uuid.UUID) error {
	result := r.db.
		WithContext(ctx).
		Model(&model.Counter{}).
		Where("id = ?", counterID).
		Update("current_employee_id", employeeID)

	if result.Error != nil {
		return fmt.Errorf("update counter employee: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrCounterNotFound, counterID)
	}

	return nil
}

func (r *CounterRepoImpl) UpdateCounterCustomer(ctx context.Context, counterID uuid.UUID, queueID *uuid.UUID) error {
	result := r.db.
		WithContext(ctx).
		Model(&model.Counter{}).
		Where("id = ?", counterID).
		Update("current_queue_id", queueID)

	if result.Error != nil {
		return fmt.Errorf("update counter customer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrCounterNotFound, counterID)
	}

	return nil
}

func (r *CounterRepoImpl) GetCounter(ctx context.Context, id uuid.UUID) (*domain.Counter, error) {
	var record model.Counter
	err := r.db.
		WithContext(ctx).
		Where("id = ?", id).
		Take(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrCounterNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get counter: %w", err)
	}

	return toDomainCounter(&record), nil
}

func (r *CounterRepoImpl) DeleteCounter(ctx context.Context, id uuid.UUID) error {
	result := r.db.
		WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.Counter{})

	if result.Error != nil {
		return fmt.Errorf("delete counter: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrCounterNotFound, id)
	}

	return nil
}
