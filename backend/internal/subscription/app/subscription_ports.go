package app

import (
	"QueueLite/internal/subscription/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type SubscriptionRepo interface {
	GetAllSubscription(ctx context.Context, cursor *domain.SubscriptionCursor, limit int) ([]domain.Subscription, *domain.SubscriptionCursor, error)
	CreateSubscription(ctx context.Context, subscription *domain.Subscription, businessPlan *domain.BusinessPlan, userPlan *domain.UserPlan) (*domain.Subscription, error)
	GetSubscription(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	UpdateSubscription(ctx context.Context, id uuid.UUID, businessPlanID *uuid.UUID, userPlanID *uuid.UUID) error
	UpdateSubscriptionTime(ctx context.Context, id uuid.UUID, startTime time.Time, endTime *time.Time) error
	ActivateUserSubscription(ctx context.Context, id uuid.UUID) error
	DeactivateUserSubscription(ctx context.Context, id uuid.UUID) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error

	GetBusinessSubscriptionInfo(ctx context.Context, businessID uuid.UUID) (*domain.BusinessSubscriptionInfo, error)
	GetUserSubscriptionInfo(ctx context.Context, userID uuid.UUID) (*domain.UserSubscriptionInfo, error)
	UseUserSubscription(ctx context.Context, userID uuid.UUID, businessID uuid.UUID) (*domain.UseUserSubscriptionResult, error)
	AddUserSlot(ctx context.Context, userID uuid.UUID, amount int) error
	DecreaseBusinessCapacity(ctx context.Context, businessID uuid.UUID) error
}
