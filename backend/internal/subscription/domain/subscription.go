package domain

import (
	"time"

	"github.com/google/uuid"
)

type BusinessPlanType string

const (
	BusinessPlanTypeFree BusinessPlanType = "free"
	BusinessPlanTypePlus BusinessPlanType = "plus"
	BusinessPlanTypePro  BusinessPlanType = "pro"
	BusinessPlanTypeMax  BusinessPlanType = "max"
)

type UserPlanType string

const (
	UserPlanTypeStandard UserPlanType = "standard"
	UserPlanTypePremium  UserPlanType = "premium"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
)

type SubscriptionType string

const (
	SubscriptionTypeBusiness SubscriptionType = "business"
	SubscriptionTypeUser     SubscriptionType = "user"
)

type Subscription struct {
	ID             uuid.UUID
	BusinessID     *uuid.UUID
	UserID         *uuid.UUID
	BusinessPlanID *uuid.UUID
	UserPlanID     *uuid.UUID
	Type           SubscriptionType
	StartDate      time.Time
	EndDate        *time.Time
	Status         SubscriptionStatus
	CreatedAt      time.Time
}

type BusinessPlan struct {
	ID                  uuid.UUID
	BusinessPlanType    BusinessPlanType
	Description         string
	Price               int64
	Capacity            int
	Analysis            bool
	Insight             bool
	PrioritySupport     bool
	LastCapacityResetAt time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UserPlan struct {
	ID               uuid.UUID
	UserPlanType     UserPlanType
	Name             string
	Description      string
	Price            int64
	Slots            int
	LastSlotsResetAt time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SubscriptionPlan struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type BusinessSubscriptionInfo struct {
	Subscription Subscription
	BusinessPlan BusinessPlan
}

type UserSubscriptionInfo struct {
	Subscription Subscription
	UserPlan     UserPlan
}

type UseUserSubscriptionResult struct {
	Success bool
	Message string
}

type SubscriptionCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}
