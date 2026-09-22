package outbound

import (
	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/subscription/app"
	"QueueLite/internal/subscription/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepoImpl struct {
	db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) *SubscriptionRepoImpl {
	return &SubscriptionRepoImpl{db: db}
}

func (r *SubscriptionRepoImpl) CreatePlan(ctx context.Context, plan *domain.SubscriptionPlan) (*domain.SubscriptionPlan, error) {
	record := toSubscriptionPlanRecord(plan)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create subscription plan: %w", err)
	}
	return toDomainSubscriptionPlan(&record), nil
}

func (r *SubscriptionRepoImpl) UpdatePlanName(ctx context.Context, id uuid.UUID, name string) error {
	return r.updatePlanColumns(ctx, id, map[string]any{"name": name})
}

func (r *SubscriptionRepoImpl) UpdatePlanDescription(ctx context.Context, id uuid.UUID, description string) error {
	return r.updatePlanColumns(ctx, id, map[string]any{"description": description})
}

func (r *SubscriptionRepoImpl) GetPlan(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	var record model.SubscriptionPlan
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrSubscriptionPlanNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription plan: %w", err)
	}
	return toDomainSubscriptionPlan(&record), nil
}

func (r *SubscriptionRepoImpl) DeletePlan(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("subscription_plan_id = ?", id).Delete(&model.Subscription{}).Error; err != nil {
			return fmt.Errorf("delete plan subscriptions: %w", err)
		}

		result := tx.Where("id = ?", id).Delete(&model.SubscriptionPlan{})
		if result.Error != nil {
			return fmt.Errorf("delete subscription plan: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("%w: %s", app.ErrSubscriptionPlanNotFound, id)
		}
		return nil
	})
}

func (r *SubscriptionRepoImpl) GetAllSubscription(ctx context.Context, cursor *domain.SubscriptionCursor, limit int) ([]domain.Subscription, *domain.SubscriptionCursor, error) {
	var records []model.Subscription
	query := r.db.WithContext(ctx).Order("created_at DESC").Order("id DESC").Limit(limit)
	if cursor != nil {
		query = query.Where("created_at < ? OR (created_at = ? AND id < ?)", cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, nil, fmt.Errorf("get subscriptions: %w", err)
	}

	var nextCursor *domain.SubscriptionCursor
	if len(records) > 0 {
		last := records[len(records)-1]
		nextCursor = &domain.SubscriptionCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	return toDomainSubscriptions(records), nextCursor, nil
}

func (r *SubscriptionRepoImpl) CreateSubscription(ctx context.Context, subscription *domain.Subscription) (*domain.Subscription, error) {
	record := toSubscriptionRecord(subscription)
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return toDomainSubscription(&record), nil
}

func (r *SubscriptionRepoImpl) GetSubscription(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	var record model.Subscription
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	return toDomainSubscription(&record), nil
}

func (r *SubscriptionRepoImpl) UpdateSubscription(ctx context.Context, id uuid.UUID, planID uuid.UUID) error {
	return r.updateSubscriptionColumns(ctx, id, map[string]any{"subscription_plan_id": planID})
}

func (r *SubscriptionRepoImpl) UpdateSubscriptionTime(ctx context.Context, id uuid.UUID, startTime time.Time, endTime *time.Time) error {
	return r.updateSubscriptionColumns(ctx, id, map[string]any{"start_date": startTime, "end_date": endTime})
}

func (r *SubscriptionRepoImpl) ActivateUserSubscription(ctx context.Context, id uuid.UUID) error {
	return r.updateSubscriptionColumns(ctx, id, map[string]any{"status": model.SubscriptionStatusActive})
}

func (r *SubscriptionRepoImpl) DeactivateUserSubscription(ctx context.Context, id uuid.UUID) error {
	return r.updateSubscriptionColumns(ctx, id, map[string]any{"status": model.SubscriptionStatusInactive})
}

func (r *SubscriptionRepoImpl) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Subscription{})
	if result.Error != nil {
		return fmt.Errorf("delete subscription: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, id)
	}
	return nil
}

func (r *SubscriptionRepoImpl) updatePlanColumns(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	result := r.db.WithContext(ctx).Model(&model.SubscriptionPlan{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update subscription plan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrSubscriptionPlanNotFound, id)
	}
	return nil
}

func (r *SubscriptionRepoImpl) updateSubscriptionColumns(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	result := r.db.WithContext(ctx).Model(&model.Subscription{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update subscription: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, id)
	}
	return nil
}
