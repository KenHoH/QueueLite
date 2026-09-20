package model

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	// Unique BusinessID makes Business ↔ Subscription one-to-one.
	BusinessID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"businessId"`

	SubscriptionPlanID uuid.UUID `gorm:"type:uuid;not null;index" json:"subscriptionPlanId"`

	Type SubscriptionType `gorm:"type:varchar(20);not null" json:"type"`

	StartDate time.Time  `gorm:"not null" json:"startDate"`
	EndDate   *time.Time `json:"endDate,omitempty"`

	Status SubscriptionStatus `gorm:"type:varchar(20);not null;default:active" json:"status"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Business Business         `gorm:"foreignKey:BusinessID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Plan     SubscriptionPlan `gorm:"foreignKey:SubscriptionPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"plan,omitempty"`
}
