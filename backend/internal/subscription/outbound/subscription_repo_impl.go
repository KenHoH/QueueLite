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
	"gorm.io/gorm/clause"
)

type SubscriptionRepoImpl struct {
	db *gorm.DB
}

func NewSubscriptionRepo(db *gorm.DB) *SubscriptionRepoImpl {
	return &SubscriptionRepoImpl{db: db}
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

func (r *SubscriptionRepoImpl) CreateSubscription(ctx context.Context, subscription *domain.Subscription, businessPlan *domain.BusinessPlan, userPlan *domain.UserPlan) (*domain.Subscription, error) {
	var created model.Subscription
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record := toSubscriptionRecord(subscription)
		if businessPlan != nil {
			planRecord := toBusinessPlanRecord(businessPlan)
			if err := tx.Create(&planRecord).Error; err != nil {
				return fmt.Errorf("create business plan: %w", err)
			}
			record.BusinessPlanID = &planRecord.ID
		}
		if userPlan != nil {
			planRecord := toUserPlanRecord(userPlan)
			if err := tx.Create(&planRecord).Error; err != nil {
				return fmt.Errorf("create user plan: %w", err)
			}
			record.UserPlanID = &planRecord.ID
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("create subscription: %w", err)
		}
		created = record
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toDomainSubscription(&created), nil
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

func (r *SubscriptionRepoImpl) UpdateSubscription(ctx context.Context, id uuid.UUID, businessPlanID *uuid.UUID, userPlanID *uuid.UUID) error {
	updates := map[string]any{}
	if businessPlanID != nil {
		updates["business_plan_id"] = businessPlanID
		updates["user_plan_id"] = nil
	}
	if userPlanID != nil {
		updates["user_plan_id"] = userPlanID
		updates["business_plan_id"] = nil
	}
	return r.updateSubscriptionColumns(ctx, id, updates)
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

func (r *SubscriptionRepoImpl) GetBusinessSubscriptionInfo(ctx context.Context, businessID uuid.UUID) (*domain.BusinessSubscriptionInfo, error) {
	var record model.Subscription
	err := r.db.WithContext(ctx).
		Preload("BusinessPlan").
		Where("business_id = ? AND type = ?", businessID, model.SubscriptionTypeBusiness).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, businessID)
	}
	if err != nil {
		return nil, fmt.Errorf("get business subscription info: %w", err)
	}
	if record.BusinessPlan == nil {
		return nil, fmt.Errorf("%w: %s", app.ErrBusinessPlanNotFound, businessID)
	}
	plan := *record.BusinessPlan
	if resetBusinessCapacityIfNeeded(&plan, time.Now()) {
		if err := r.db.WithContext(ctx).Save(&plan).Error; err != nil {
			return nil, fmt.Errorf("reset business capacity: %w", err)
		}
	}
	return &domain.BusinessSubscriptionInfo{Subscription: *toDomainSubscription(&record), BusinessPlan: *toDomainBusinessPlan(&plan)}, nil
}

func (r *SubscriptionRepoImpl) GetUserSubscriptionInfo(ctx context.Context, userID uuid.UUID) (*domain.UserSubscriptionInfo, error) {
	var record model.Subscription
	err := r.db.WithContext(ctx).
		Preload("UserPlan").
		Where("user_id = ? AND type = ?", userID, model.SubscriptionTypeUser).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("get user subscription info: %w", err)
	}
	if record.UserPlan == nil {
		return nil, fmt.Errorf("%w: %s", app.ErrUserPlanNotFound, userID)
	}
	plan := *record.UserPlan
	if resetUserSlotsIfNeeded(&plan, time.Now()) {
		if err := r.db.WithContext(ctx).Save(&plan).Error; err != nil {
			return nil, fmt.Errorf("reset user slots: %w", err)
		}
	}
	return &domain.UserSubscriptionInfo{Subscription: *toDomainSubscription(&record), UserPlan: *toDomainUserPlan(&plan)}, nil
}

func (r *SubscriptionRepoImpl) UseUserSubscription(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.UseUserSubscriptionResult, error) {
	result := &domain.UseUserSubscriptionResult{Success: false}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		var userSub model.Subscription
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("UserPlan").Where("user_id = ? AND type = ?", userID, model.SubscriptionTypeUser).Take(&userSub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Message = "user subscription not found"
				return nil
			}
			return fmt.Errorf("get user subscription: %w", err)
		}
		if userSub.UserPlan == nil {
			result.Message = "user plan not found"
			return nil
		}
		if resetUserSlotsIfNeeded(userSub.UserPlan, now) {
			if err := tx.Save(userSub.UserPlan).Error; err != nil {
				return fmt.Errorf("reset user slots: %w", err)
			}
		}
		if !subscriptionActive(userSub, now) {
			result.Message = inactiveMessage("user", userSub, now)
			return nil
		}
		if userSub.UserPlan.Slots <= 0 {
			result.Message = "no user priority slots available"
			return nil
		}

		var businessSub model.Subscription
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("BusinessPlan").Where("business_id = ? AND type = ?", businessID, model.SubscriptionTypeBusiness).Take(&businessSub).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Message = "business subscription not found"
				return nil
			}
			return fmt.Errorf("get business subscription: %w", err)
		}
		if businessSub.BusinessPlan == nil {
			result.Message = "business plan not found"
			return nil
		}
		if resetBusinessCapacityIfNeeded(businessSub.BusinessPlan, now) {
			if err := tx.Save(businessSub.BusinessPlan).Error; err != nil {
				return fmt.Errorf("reset business capacity: %w", err)
			}
		}
		if !subscriptionActive(businessSub, now) {
			result.Message = inactiveMessage("business", businessSub, now)
			return nil
		}
		if !businessSub.BusinessPlan.PrioritySupport {
			result.Message = "business does not support priority queue"
			return nil
		}
		if businessSub.BusinessPlan.Capacity <= 0 {
			result.Message = "business queue capacity is unavailable"
			return nil
		}

		userSub.UserPlan.Slots--
		businessSub.BusinessPlan.Capacity--
		if err := tx.Save(userSub.UserPlan).Error; err != nil {
			return fmt.Errorf("decrease user slot: %w", err)
		}
		if err := tx.Save(businessSub.BusinessPlan).Error; err != nil {
			return fmt.Errorf("decrease business capacity: %w", err)
		}
		result.Success = true
		result.Message = "priority subscription used"
		return nil
	})
	return result, err
}

func (r *SubscriptionRepoImpl) AddUserSlot(ctx context.Context, userID uuid.UUID, amount int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record model.Subscription
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("UserPlan").Where("user_id = ? AND type = ?", userID, model.SubscriptionTypeUser).Take(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, userID)
			}
			return fmt.Errorf("get user subscription: %w", err)
		}
		if record.UserPlan == nil {
			return fmt.Errorf("%w: %s", app.ErrUserPlanNotFound, userID)
		}
		record.UserPlan.Slots += amount
		if err := tx.Save(record.UserPlan).Error; err != nil {
			return fmt.Errorf("add user slot: %w", err)
		}
		return nil
	})
}

func (r *SubscriptionRepoImpl) DecreaseBusinessCapacity(ctx context.Context, businessID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record model.Subscription
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Preload("BusinessPlan").Where("business_id = ? AND type = ?", businessID, model.SubscriptionTypeBusiness).Take(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, businessID)
			}
			return fmt.Errorf("get business subscription: %w", err)
		}
		if record.BusinessPlan == nil {
			return fmt.Errorf("%w: %s", app.ErrBusinessPlanNotFound, businessID)
		}
		if resetBusinessCapacityIfNeeded(record.BusinessPlan, time.Now()) {
			if err := tx.Save(record.BusinessPlan).Error; err != nil {
				return fmt.Errorf("reset business capacity: %w", err)
			}
		}
		if record.BusinessPlan.Capacity <= 0 {
			return nil
		}
		record.BusinessPlan.Capacity--
		if err := tx.Save(record.BusinessPlan).Error; err != nil {
			return fmt.Errorf("decrease business capacity: %w", err)
		}
		return nil
	})
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

func subscriptionActive(subscription model.Subscription, now time.Time) bool {
	if subscription.Status != model.SubscriptionStatusActive {
		return false
	}
	return subscription.EndDate == nil || subscription.EndDate.After(now)
}

func inactiveMessage(owner string, subscription model.Subscription, now time.Time) string {
	if subscription.EndDate != nil && !subscription.EndDate.After(now) {
		return owner + " subscription is expired"
	}
	return owner + " subscription is inactive"
}

func resetBusinessCapacityIfNeeded(plan *model.BusinessPlan, now time.Time) bool {
	if sameDay(plan.LastCapacityResetAt, now) {
		return false
	}
	plan.Capacity = defaultBusinessCapacity(model.BusinessPlanType(plan.BusinessPlanType))
	plan.LastCapacityResetAt = now
	return true
}

func resetUserSlotsIfNeeded(plan *model.UserPlan, now time.Time) bool {
	if sameMonth(plan.LastSlotsResetAt, now) {
		return false
	}
	plan.Slots = defaultUserSlots(model.UserPlanType(plan.UserPlanType))
	plan.LastSlotsResetAt = now
	return true
}

func sameDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func sameMonth(a time.Time, b time.Time) bool {
	ay, am, _ := a.Date()
	by, bm, _ := b.Date()
	return ay == by && am == bm
}

func defaultBusinessCapacity(planType model.BusinessPlanType) int {
	switch planType {
	case model.BusinessPlanTypePlus:
		return 500
	case model.BusinessPlanTypePro:
		return 2000
	case model.BusinessPlanTypeMax:
		return 10000
	default:
		return 100
	}
}

func defaultUserSlots(planType model.UserPlanType) int {
	if planType == model.UserPlanTypePremium {
		return 4
	}
	return 0
}
