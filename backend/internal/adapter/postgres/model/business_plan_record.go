package model

import (
	"time"

	"github.com/google/uuid"
)

type BusinessPlan struct {
	ID               uuid.UUID        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusinessPlanType BusinessPlanType `gorm:"type:varchar(20);not null;default:free;index" json:"businessPlanType"`
	Description      string           `gorm:"type:text" json:"description"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Price               int64     `gorm:"type:bigint;not null;default:0" json:"price"`
	Capacity            int       `gorm:"not null;default:0" json:"capacity"`
	Analysis            bool      `gorm:"not null;default:false" json:"analysis"`
	Insight             bool      `gorm:"not null;default:false" json:"insight"`
	PrioritySupport     bool      `gorm:"not null;default:false" json:"prioritySupport"`
	LastCapacityResetAt time.Time `json:"lastCapacityResetAt"`

	Subscriptions []Subscription `gorm:"foreignKey:BusinessPlanID" json:"subscriptions,omitempty"`
}
