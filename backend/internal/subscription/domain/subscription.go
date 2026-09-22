package domain

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

type SubscriptionType string

const (
	SubscriptionTypeMonthly SubscriptionType = "business"
	SubscriptionTypeYearly  SubscriptionType = "person"
)

type Subscription struct {
	ID                 uuid.UUID
	BusinessID         uuid.UUID
	SubscriptionPlanID uuid.UUID
	Type               SubscriptionType
	StartDate          time.Time
	EndDate            *time.Time
	Status             SubscriptionStatus
	CreatedAt          time.Time
}

type SubscriptionPlan struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type SubscriptionCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}
