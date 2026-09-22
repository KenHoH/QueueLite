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

func (s *SubscriptionService) CreatePlan(ctx context.Context, plan domain.SubscriptionPlan) (*domain.SubscriptionPlan, error) {
	plan.Name = strings.TrimSpace(plan.Name)
	plan.Description = strings.TrimSpace(plan.Description)
	if plan.Name == "" {
		return nil, apperror.New(apperror.KindInvalid, "PLAN_NAME_REQUIRED", "subscription plan name is required")
	}
	record, err := s.repo.CreatePlan(ctx, &plan)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "CREATE_SUBSCRIPTION_PLAN_ERROR", "failed to create subscription plan", err)
	}
	return record, nil
}

func (s *SubscriptionService) UpdatePlanName(ctx context.Context, id uuid.UUID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return apperror.New(apperror.KindInvalid, "PLAN_NAME_REQUIRED", "subscription plan name is required")
	}
	return s.handlePlanUpdateError(s.repo.UpdatePlanName(ctx, id, name), "UPDATE_SUBSCRIPTION_PLAN_NAME_ERROR")
}

func (s *SubscriptionService) UpdatePlanDescription(ctx context.Context, id uuid.UUID, description string) error {
	description = strings.TrimSpace(description)
	return s.handlePlanUpdateError(s.repo.UpdatePlanDescription(ctx, id, description), "UPDATE_SUBSCRIPTION_PLAN_DESCRIPTION_ERROR")
}

func (s *SubscriptionService) GetPlan(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	plan, err := s.repo.GetPlan(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSubscriptionPlanNotFound) {
			return nil, apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_PLAN_NOT_FOUND", "subscription plan not found", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "GET_SUBSCRIPTION_PLAN_ERROR", "failed to get subscription plan", err)
	}
	return plan, nil
}

func (s *SubscriptionService) DeletePlan(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeletePlan(ctx, id); err != nil {
		if errors.Is(err, ErrSubscriptionPlanNotFound) {
			return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_PLAN_NOT_FOUND", "subscription plan not found", err)
		}
		return apperror.Wrap(apperror.KindInternal, "DELETE_SUBSCRIPTION_PLAN_ERROR", "failed to delete subscription plan", err)
	}
	return nil
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

func (s *SubscriptionService) CreateSubscription(ctx context.Context, subscription domain.Subscription) (*domain.Subscription, error) {
	if subscription.BusinessID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "BUSINESS_ID_REQUIRED", "business id is required")
	}
	if subscription.SubscriptionPlanID == uuid.Nil {
		return nil, apperror.New(apperror.KindInvalid, "SUBSCRIPTION_PLAN_ID_REQUIRED", "subscription plan id is required")
	}
	if subscription.Type == "" {
		return nil, apperror.New(apperror.KindInvalid, "SUBSCRIPTION_TYPE_REQUIRED", "subscription type is required")
	}
	if !isValidSubscriptionType(subscription.Type) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_SUBSCRIPTION_TYPE", "invalid subscription type")
	}
	if subscription.StartDate.IsZero() {
		return nil, apperror.New(apperror.KindInvalid, "SUBSCRIPTION_START_TIME_REQUIRED", "subscription start time is required")
	}
	if subscription.Status == "" {
		subscription.Status = domain.SubscriptionStatusActive
	}
	if !isValidSubscriptionStatus(subscription.Status) {
		return nil, apperror.New(apperror.KindInvalid, "INVALID_SUBSCRIPTION_STATUS", "invalid subscription status")
	}

	record, err := s.repo.CreateSubscription(ctx, &subscription)
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

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, id uuid.UUID, planID uuid.UUID) error {
	if planID == uuid.Nil {
		return apperror.New(apperror.KindInvalid, "SUBSCRIPTION_PLAN_ID_REQUIRED", "subscription plan id is required")
	}
	return s.handleSubscriptionUpdateError(s.repo.UpdateSubscription(ctx, id, planID), "UPDATE_SUBSCRIPTION_ERROR")
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

func (s *SubscriptionService) handlePlanUpdateError(err error, code string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrSubscriptionPlanNotFound) {
		return apperror.Wrap(apperror.KindNotFound, "SUBSCRIPTION_PLAN_NOT_FOUND", "subscription plan not found", err)
	}
	return apperror.Wrap(apperror.KindInternal, code, "failed to update subscription plan", err)
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
	case domain.SubscriptionTypeMonthly, domain.SubscriptionTypeYearly:
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
