package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/subscription/domain"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type SubscriptionService struct {
	repo SubscriptionRepo
}

func NewSubscriptionService(repo SubscriptionRepo) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

func (s *SubscriptionService) GetAllSubscription(ctx context.Context, cursor *domain.SubscriptionCursor, limit int) ([]domain.Subscription, *domain.SubscriptionCursor, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	subscriptions, nextCursor, err := s.repo.GetAllSubscription(ctx, cursor, limit)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.KindInternal, "GET_ALL_SUBSCRIPTION_ERROR", "failed to get subscriptions", err)
	}
	return subscriptions, nextCursor, nil
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, subscription domain.Subscription, businessPlanType domain.BusinessPlanType, userPlanType domain.UserPlanType) (*domain.Subscription, error) {

	if subscription.Type == "" {
		return nil, apperror.New(apperror.KindInvalid, "SUBSCRIPTION_TYPE_REQUIRED", "subscription type is required")
	}
	if !IsValidSubscriptionType(subscription.Type) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_SUBSCRIPTION_TYPE", "invalid subscription type")
	}

	// default value for new subscription Time
	subscription.StartDate = time.Now()
	subscription.Status = domain.SubscriptionStatusActive

	if !IsValidSubscriptionStatus(subscription.Status) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_SUBSCRIPTION_STATUS", "invalid subscription status")
	}

	var businessPlan *domain.BusinessPlan
	var userPlan *domain.UserPlan
	now := time.Now()

	switch subscription.Type {
	case domain.SubscriptionTypeBusiness:
		if subscription.BusinessID == nil || *subscription.BusinessID == uuid.Nil {
			return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
		}
		if businessPlanType == "" {
			businessPlanType = domain.BusinessPlanTypeFree
		}
		if !IsValidBusinessPlanType(businessPlanType) {
			return nil, apperror.New(apperror.KindInvalid, "INVALID_BUSINESS_PLAN_TYPE", "invalid business plan type")
		}
		plan := DefaultBusinessPlan(businessPlanType, now)
		businessPlan = &plan
	case domain.SubscriptionTypeUser:
		if subscription.UserID == nil || *subscription.UserID == uuid.Nil {
			return nil, apperror.New(apperror.KindInvalid, "USER_ID_REQUIRED", "user id is required")
		}
		if userPlanType == "" {
			userPlanType = domain.UserPlanTypeStandard
		}
		if !IsValidUserPlanType(userPlanType) {
			return nil, apperror.New(apperror.KindInvalid, "INVALID_USER_PLAN_TYPE", "invalid user plan type")
		}
		plan := DefaultUserPlan(userPlanType, now)
		userPlan = &plan
	}

	record, err := s.repo.CreateSubscription(ctx, &subscription, businessPlan, userPlan)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CREATE_SUBSCRIPTION_ERROR", "failed to create subscription", err)
	}
	return record, nil
}

func (s *SubscriptionService) CreateDefaultUserSubscriptionPlan(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return s.CreateSubscription(ctx, domain.Subscription{
		UserID: &userID,
		Type:   domain.SubscriptionTypeUser,
		Status: domain.SubscriptionStatusActive},
		domain.BusinessPlanTypeFree,
		domain.UserPlanTypeStandard,
	)
}

func (s *SubscriptionService) CreateDefaultBusinessSubscriptionPlan(ctx context.Context, businessID uuid.UUID) (*domain.Subscription, error) {
	return s.CreateSubscription(ctx, domain.Subscription{
		BusinessID: &businessID,
		Type:       domain.SubscriptionTypeBusiness,
		Status:     domain.SubscriptionStatusActive},
		domain.BusinessPlanTypeFree,
		domain.UserPlanTypeStandard,
	)
}

func (s *SubscriptionService) GetSubscription(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	subscription, err := s.repo.GetSubscription(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_SUBSCRIPTION_ERROR", "failed to get subscription", err)
	}
	return subscription, nil
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, id uuid.UUID, businessPlanID *uuid.UUID, userPlanID *uuid.UUID) error {
	if businessPlanID == nil && userPlanID == nil {
		return apperror.New(apperror.KindInvalid, "SUBSCRIPTION_PLAN_ID_REQUIRED", "subscription plan id is required")
	}
	if err := s.repo.UpdateSubscription(ctx, id, businessPlanID, userPlanID); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_SUBSCRIPTION_ERROR", "failed to update subscription", err)
	}
	return nil
}

func (s *SubscriptionService) UpdateSubscriptionTime(ctx context.Context, id uuid.UUID, startTime time.Time, endTime *time.Time) error {
	if startTime.IsZero() {
		return apperror.New(apperror.KindInvalid, "SUBSCRIPTION_START_TIME_REQUIRED", "subscription start time is required")
	}
	if endTime != nil && !endTime.After(startTime) {
		return apperror.New(apperror.KindInvalid, "INVALID_SUBSCRIPTION_END_TIME", "subscription end time must be after start time")
	}
	if err := s.repo.UpdateSubscriptionTime(ctx, id, startTime, endTime); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "UPDATE_SUBSCRIPTION_TIME_ERROR", "failed to update subscription time", err)
	}
	return nil
}

func (s *SubscriptionService) ActivateUserSubscription(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.ActivateUserSubscription(ctx, id); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "ACTIVATE_SUBSCRIPTION_ERROR", "failed to activate subscription", err)
	}
	return nil
}

func (s *SubscriptionService) DeactivateUserSubscription(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeactivateUserSubscription(ctx, id); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DEACTIVATE_SUBSCRIPTION_ERROR", "failed to deactivate subscription", err)
	}
	return nil
}

func (s *SubscriptionService) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteSubscription(ctx, id); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_SUBSCRIPTION_ERROR", "failed to delete subscription", err)
	}
	return nil
}

func (s *SubscriptionService) GetBusinessSubscriptionInfo(ctx context.Context, businessID uuid.UUID) (*domain.BusinessSubscriptionInfo, error) {
	if businessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	info, err := s.repo.GetBusinessSubscriptionInfo(ctx, businessID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "business subscription not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_BUSINESS_SUBSCRIPTION_INFO_ERROR", "failed to get business subscription info", err)
	}
	return info, nil
}

func (s *SubscriptionService) GetUserSubscriptionInfo(ctx context.Context, userID uuid.UUID) (*domain.UserSubscriptionInfo, error) {
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "USER_ID_REQUIRED", "user id is required")
	}
	info, err := s.repo.GetUserSubscriptionInfo(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "user subscription not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_USER_SUBSCRIPTION_INFO_ERROR", "failed to get user subscription info", err)
	}
	return info, nil
}

func (s *SubscriptionService) UseUserSubscription(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.UseUserSubscriptionResult, error) {
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "USER_ID_REQUIRED", "user id is required")
	}
	if businessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	result, err := s.repo.UseUserSubscription(ctx, userID, businessID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "USE_USER_SUBSCRIPTION_ERROR", "failed to use user subscription", err)
	}
	return result, nil
}

func (s *SubscriptionService) AddUserSlot(ctx context.Context, userID uuid.UUID, amount int) error {
	if userID == uuid.Nil {
		return apperror.New(apperror.KindInvalid, "USER_ID_REQUIRED", "user id is required")
	}
	if amount <= 0 {
		return apperror.New(apperror.KindInvalid, "INVALID_SLOT_AMOUNT", "slot amount must be positive")
	}
	if err := s.repo.AddUserSlot(ctx, userID, amount); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "user subscription not found", err)
		}
		if errors.Is(err, ErrUserPlanNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "USER_PLAN_NOT_FOUND", "user plan not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "ADD_USER_SLOT_ERROR", "failed to add user slot", err)
	}
	return nil
}

func (s *SubscriptionService) CheckUserPrioritySlot(ctx context.Context, userID uuid.UUID) error {
	info, err := s.GetUserSubscriptionInfo(ctx, userID)
	if err != nil {
		return err
	}
	if info.UserPlan.Slots <= 0 {
		return apperror.New(apperror.KindInvalid, "NO_PRIORITY_SLOT", "there's no priority token left")
	}
	return nil
}

func (s *SubscriptionService) DecreaseBusinessCapacity(ctx context.Context, businessID uuid.UUID) error {
	if businessID == uuid.Nil {
		return apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if err := s.CheckBusinessQueueQuota(ctx, businessID); err != nil {
		return err
	}
	if err := s.repo.DecreaseBusinessCapacity(ctx, businessID); err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "business subscription not found", err)
		}
		if errors.Is(err, ErrBusinessPlanNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "BUSINESS_PLAN_NOT_FOUND", "business plan not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DECREASE_BUSINESS_CAPACITY_ERROR", "failed to decrease business capacity", err)
	}
	return nil
}

func (s *SubscriptionService) CheckBusinessQueueQuota(ctx context.Context, businessID uuid.UUID) error {
	info, err := s.GetBusinessSubscriptionInfo(ctx, businessID)
	if err != nil {
		return err
	}
	if info.BusinessPlan.Capacity <= 0 {
		return apperror.New(apperror.KindInvalid, "QUEUE_FULL", "queue is full")
	}
	return nil
}
