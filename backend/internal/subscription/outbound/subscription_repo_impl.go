package outbound

import (
	"context"
	"errors"
	"fmt"
	"time"

	"QueueLite/internal/adapter/postgres/model"
	"QueueLite/internal/subscription/app"
	"QueueLite/internal/subscription/domain"

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

func (r *SubscriptionRepoImpl) UpdateUserSubscriptionPlanType(ctx context.Context, userID uuid.UUID, userPlan domain.UserPlan) error {
	return r.withLockedUserPlan(ctx, userID, func(tx *gorm.DB, plan *model.UserPlan) error {
		plan.UserPlanType = model.UserPlanType(userPlan.UserPlanType)
		plan.Name = userPlan.Name
		plan.Description = userPlan.Description
		plan.Price = userPlan.Price
		plan.Slots = userPlan.Slots
		plan.LastSlotsResetAt = userPlan.LastSlotsResetAt
		if err := tx.Save(plan).Error; err != nil {
			return fmt.Errorf("update user plan: %w", err)
		}
		return nil
	})
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
	if !sameDay(plan.LastCapacityResetAt, time.Now()) {
		if err := r.withLockedBusinessPlan(ctx, businessID, func(tx *gorm.DB, lockedPlan *model.BusinessPlan) error {
			if resetBusinessCapacityIfNeeded(lockedPlan, time.Now()) {
				if err := tx.Save(lockedPlan).Error; err != nil {
					return fmt.Errorf("reset business capacity: %w", err)
				}
			}
			plan = *lockedPlan
			return nil
		}); err != nil {
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
	if !sameMonth(plan.LastSlotsResetAt, time.Now()) {
		if err := r.withLockedUserPlan(ctx, userID, func(tx *gorm.DB, lockedPlan *model.UserPlan) error {
			if resetUserSlotsIfNeeded(lockedPlan, time.Now()) {
				if err := tx.Save(lockedPlan).Error; err != nil {
					return fmt.Errorf("reset user slots: %w", err)
				}
			}
			plan = *lockedPlan
			return nil
		}); err != nil {
			return nil, fmt.Errorf("reset user slots: %w", err)
		}
	}
	return &domain.UserSubscriptionInfo{Subscription: *toDomainSubscription(&record), UserPlan: *toDomainUserPlan(&plan)}, nil
}

func (r *SubscriptionRepoImpl) UseUserSubscription(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.UseUserSubscriptionResult, error) {
	result := &domain.UseUserSubscriptionResult{Success: false}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		userSub, err := loadLockedUserSubscription(tx, userID)
		if err != nil {
			if errors.Is(err, app.ErrSubscriptionNotFound) {
				result.Message = "user subscription not found"
				return nil
			}
			if errors.Is(err, app.ErrUserPlanNotFound) {
				result.Message = "user plan not found"
				return nil
			}
			return err
		}
		if resetUserSlotsIfNeeded(userSub.UserPlan, now) {
			if err := tx.Save(userSub.UserPlan).Error; err != nil {
				return fmt.Errorf("reset user slots: %w", err)
			}
		}
		if !subscriptionActive(*userSub, now) {
			result.Message = inactiveMessage("user", *userSub, now)
			return nil
		}
		if userSub.UserPlan.Slots <= 0 {
			result.Message = "no user priority slots available"
			return nil
		}

		businessSub, err := loadLockedBusinessSubscription(tx, businessID)
		if err != nil {
			if errors.Is(err, app.ErrSubscriptionNotFound) {
				result.Message = "business subscription not found"
				return nil
			}
			if errors.Is(err, app.ErrBusinessPlanNotFound) {
				result.Message = "business plan not found"
				return nil
			}
			return err
		}
		if resetBusinessCapacityIfNeeded(businessSub.BusinessPlan, now) {
			if err := tx.Save(businessSub.BusinessPlan).Error; err != nil {
				return fmt.Errorf("reset business capacity: %w", err)
			}
		}
		if !subscriptionActive(*businessSub, now) {
			result.Message = inactiveMessage("business", *businessSub, now)
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
	return r.updateUserSlots(ctx, userID, amount)
}

func (r *SubscriptionRepoImpl) DecreaseUserSlot(ctx context.Context, userID uuid.UUID) error {
	return r.updateUserSlots(ctx, userID, -1)
}

func (r *SubscriptionRepoImpl) updateUserSlots(ctx context.Context, userID uuid.UUID, delta int) error {
	return r.withLockedUserPlan(ctx, userID, func(tx *gorm.DB, plan *model.UserPlan) error {
		if delta < 0 && plan.Slots+delta < 0 {
			return fmt.Errorf("%w: %s", app.ErrInsufficientPrioritySlots, userID)
		}
		plan.Slots += delta
		if err := tx.Save(plan).Error; err != nil {
			return fmt.Errorf("update user slot: %w", err)
		}
		return nil
	})
}

func (r *SubscriptionRepoImpl) DecreaseBusinessCapacity(ctx context.Context, businessID uuid.UUID) error {
	return r.withLockedBusinessPlan(ctx, businessID, func(tx *gorm.DB, plan *model.BusinessPlan) error {
		if resetBusinessCapacityIfNeeded(plan, time.Now()) {
			if err := tx.Save(plan).Error; err != nil {
				return fmt.Errorf("reset business capacity: %w", err)
			}
		}
		if plan.Capacity <= 0 {
			return app.ErrBusinessQueueFull
		}
		plan.Capacity--
		if err := tx.Save(plan).Error; err != nil {
			return fmt.Errorf("decrease business capacity: %w", err)
		}
		return nil
	})
}

func (r *SubscriptionRepoImpl) IncreaseBusinessCapacity(ctx context.Context, businessID uuid.UUID) error {
	return r.withLockedBusinessPlan(ctx, businessID, func(tx *gorm.DB, plan *model.BusinessPlan) error {
		plan.Capacity++
		if err := tx.Save(plan).Error; err != nil {
			return fmt.Errorf("increase business capacity: %w", err)
		}
		return nil
	})
}

func loadLockedBusinessSubscription(tx *gorm.DB, businessID uuid.UUID) (*model.Subscription, error) {
	var record model.Subscription
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("BusinessPlan").
		Where("business_id = ? AND type = ?", businessID, model.SubscriptionTypeBusiness).
		Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, businessID)
		}
		return nil, fmt.Errorf("get business subscription: %w", err)
	}
	if record.BusinessPlan == nil {
		return nil, fmt.Errorf("%w: %s", app.ErrBusinessPlanNotFound, businessID)
	}
	return &record, nil
}

func loadLockedUserSubscription(tx *gorm.DB, userID uuid.UUID) (*model.Subscription, error) {
	var record model.Subscription
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("UserPlan").
		Where("user_id = ? AND type = ?", userID, model.SubscriptionTypeUser).
		Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", app.ErrSubscriptionNotFound, userID)
		}
		return nil, fmt.Errorf("get user subscription: %w", err)
	}
	if record.UserPlan == nil {
		return nil, fmt.Errorf("%w: %s", app.ErrUserPlanNotFound, userID)
	}
	return &record, nil
}

func (r *SubscriptionRepoImpl) withLockedBusinessPlan(ctx context.Context, businessID uuid.UUID, fn func(tx *gorm.DB, plan *model.BusinessPlan) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record, err := loadLockedBusinessSubscription(tx, businessID)
		if err != nil {
			return err
		}
		return fn(tx, record.BusinessPlan)
	})
}

func (r *SubscriptionRepoImpl) withLockedUserPlan(ctx context.Context, userID uuid.UUID, fn func(tx *gorm.DB, plan *model.UserPlan) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record, err := loadLockedUserSubscription(tx, userID)
		if err != nil {
			return err
		}
		return fn(tx, record.UserPlan)
	})
}

func (r *SubscriptionRepoImpl) ConsumeBusinessCapacityBatch(ctx context.Context, businessID uuid.UUID, amount int) error {
	if amount <= 0 {
		return nil
	}
	return r.withLockedBusinessPlan(ctx, businessID, func(tx *gorm.DB, plan *model.BusinessPlan) error {
		resetBusinessCapacityIfNeeded(plan, time.Now())
		plan.Capacity = max(plan.Capacity-amount, 0)
		if err := tx.Save(plan).Error; err != nil {
			return fmt.Errorf("consume business capacity batch: %w", err)
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
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func sameMonth(a time.Time, b time.Time) bool {
	ay, am, _ := a.UTC().Date()
	by, bm, _ := b.UTC().Date()
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
