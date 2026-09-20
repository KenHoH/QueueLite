package model

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionPlan struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string    `gorm:"type:text" json:"description"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Subscriptions []Subscription `gorm:"foreignKey:SubscriptionPlanID" json:"subscriptions,omitempty"`
}
