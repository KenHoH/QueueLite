package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/subscription/domain"
	"context"
	"errors"
	"strings"
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
	if !isValidSubscriptionType(subscription.Type) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_SUBSCRIPTION_TYPE", "invalid subscription type")
	}
	if subscription.StartDate.IsZero() {
		subscription.StartDate = time.Now()
	}
	if subscription.Status == "" {
		subscription.Status = domain.SubscriptionStatusActive
	}
	if !isValidSubscriptionStatus(subscription.Status) {
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
		if !isValidBusinessPlanType(businessPlanType) {
			return nil, apperror.New(apperror.KindInvalid, "INVALID_BUSINESS_PLAN_TYPE", "invalid business plan type")
		}
		plan := defaultBusinessPlan(businessPlanType, now)
		businessPlan = &plan
	case domain.SubscriptionTypeUser:
		if subscription.UserID == nil || *subscription.UserID == uuid.Nil {
			return nil, apperror.New(apperror.KindInvalid, "USER_ID_REQUIRED", "user id is required")
		}
		if userPlanType == "" {
			userPlanType = domain.UserPlanTypeStandard
		}
		if !isValidUserPlanType(userPlanType) {
			return nil, apperror.New(apperror.KindInvalid, "INVALID_USER_PLAN_TYPE", "invalid user plan type")
		}
		plan := defaultUserPlan(userPlanType, now)
		userPlan = &plan
	}

	record, err := s.repo.CreateSubscription(ctx, &subscription, businessPlan, userPlan)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CREATE_SUBSCRIPTION_ERROR", "failed to create subscription", err)
	}
	return record, nil
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
	return s.handleSubscriptionUpdateError(s.repo.UpdateSubscription(ctx, id, businessPlanID, userPlanID), "UPDATE_SUBSCRIPTION_ERROR")
}

func (s *SubscriptionService) UpdateSubscriptionTime(ctx context.Context, id uuid.UUID, startTime time.Time, endTime *time.Time) error {
	if startTime.IsZero() {
		return apperror.New(apperror.KindInvalid, "SUBSCRIPTION_START_TIME_REQUIRED", "subscription start time is required")
	}
	return s.handleSubscriptionUpdateError(s.repo.UpdateSubscriptionTime(ctx, id, startTime, endTime), "UPDATE_SUBSCRIPTION_TIME_ERROR")
}

func (s *SubscriptionService) ActivateUserSubscription(ctx context.Context, id uuid.UUID) error {
	return s.handleSubscriptionUpdateError(s.repo.ActivateUserSubscription(ctx, id), "ACTIVATE_SUBSCRIPTION_ERROR")
}

func (s *SubscriptionService) DeactivateUserSubscription(ctx context.Context, id uuid.UUID) error {
	return s.handleSubscriptionUpdateError(s.repo.DeactivateUserSubscription(ctx, id), "DEACTIVATE_SUBSCRIPTION_ERROR")
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
	return s.handleSubscriptionUpdateError(s.repo.AddUserSlot(ctx, userID, amount), "ADD_USER_SLOT_ERROR")
}

func (s *SubscriptionService) DecreaseBusinessCapacity(ctx context.Context, businessID uuid.UUID) error {
	if businessID == uuid.Nil {
		return apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	return s.handleSubscriptionUpdateError(s.repo.DecreaseBusinessCapacity(ctx, businessID), "DECREASE_BUSINESS_CAPACITY_ERROR")
}

func (s *SubscriptionService) handleSubscriptionUpdateError(err error, code string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSubscriptionNotFound) {
		return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_NOT_FOUND", "subscription not found", err)
	}
	return apperror.Wrap(apperror.KindInternal, code, "failed to update subscription", err)
}

func isValidSubscriptionType(value domain.SubscriptionType) bool {
	switch value {
	case domain.SubscriptionTypeBusiness, domain.SubscriptionTypeUser:
		return true
	default:
		return false
	}
}

func isValidSubscriptionStatus(value domain.SubscriptionStatus) bool {
	switch value {
	case domain.SubscriptionStatusActive, domain.SubscriptionStatusInactive:
		return true
	default:
		return false
	}
}

func isValidBusinessPlanType(value domain.BusinessPlanType) bool {
	switch value {
	case domain.BusinessPlanTypeFree, domain.BusinessPlanTypePlus, domain.BusinessPlanTypePro, domain.BusinessPlanTypeMax:
		return true
	default:
		return false
	}
}

func isValidUserPlanType(value domain.UserPlanType) bool {
	switch value {
	case domain.UserPlanTypeStandard, domain.UserPlanTypePremium:
		return true
	default:
		return false
	}
}

func defaultBusinessPlan(planType domain.BusinessPlanType, now time.Time) domain.BusinessPlan {
	plan := domain.BusinessPlan{BusinessPlanType: planType, LastCapacityResetAt: now}
	switch planType {
	case domain.BusinessPlanTypePlus:
		plan.Description = "Plus business plan"
		plan.Price = 299
		plan.Capacity = 500
		plan.Analysis = true
		plan.PrioritySupport = true
	case domain.BusinessPlanTypePro:
		plan.Description = "Pro business plan"
		plan.Price = 699
		plan.Capacity = 2000
		plan.Analysis = true
		plan.Insight = true
		plan.PrioritySupport = true
	case domain.BusinessPlanTypeMax:
		plan.Description = "Max business plan"
		plan.Price = 899
		plan.Capacity = 10000
		plan.Analysis = true
		plan.Insight = true
		plan.PrioritySupport = true
	default:
		plan.BusinessPlanType = domain.BusinessPlanTypeFree
		plan.Description = "Free business plan"
		plan.Price = 0
		plan.Capacity = 100
		plan.Analysis = true
		plan.PrioritySupport = false
	}
	return plan
}

func defaultUserPlan(planType domain.UserPlanType, now time.Time) domain.UserPlan {
	plan := domain.UserPlan{UserPlanType: planType, LastSlotsResetAt: now}
	switch planType {
	case domain.UserPlanTypePremium:
		plan.Name = "Premium"
		plan.Description = "Premium user plan"
		plan.Price = 499
		plan.Slots = 4
	default:
		plan.UserPlanType = domain.UserPlanTypeStandard
		plan.Name = "Standard"
		plan.Description = "Standard user plan"
		plan.Price = 0
		plan.Slots = 0
	}
	plan.Name = strings.TrimSpace(plan.Name)
	return plan
}
