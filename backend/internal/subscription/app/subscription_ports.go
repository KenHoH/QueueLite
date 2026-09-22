package app

import (
	"QueueLite/internal/subscription/domain"
	"context"
	"time"

	"github.com/google/uuid"
)

type SubscriptionRepo interface {
	CreatePlan(ctx context.Context, plan *domain.SubscriptionPlan) (*domain.SubscriptionPlan, error)
	UpdatePlanName(ctx context.Context, id uuid.UUID, name string) error
	UpdatePlanDescription(ctx context.Context, id uuid.UUID, description string) error
	GetPlan(ctx context.Context, id uuid.UUID) (*domain.SubscriptionPlan, error)
	DeletePlan(ctx context.Context, id uuid.UUID) error
	GetAllSubscription(ctx context.Context, cursor *domain.SubscriptionCursor, limit int) ([]domain.Subscription, *domain.SubscriptionCursor, error)
	CreateSubscription(ctx context.Context, subscription *domain.Subscription) (*domain.Subscription, error)
	GetSubscription(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	UpdateSubscription(ctx context.Context, id uuid.UUID, planID uuid.UUID) error
	UpdateSubscriptionTime(ctx context.Context, id uuid.UUID, startTime time.Time, endTime *time.Time) error
	ActivateUserSubscription(ctx context.Context, id uuid.UUID) error
	DeactivateUserSubscription(ctx context.Context, id uuid.UUID) error
	DeleteSubscription(ctx context.Context, id uuid.UUID) error
}
