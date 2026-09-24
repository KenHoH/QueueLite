package model

import (
	"time"

	"github.com/google/uuid"
)

type UserPlan struct {
	ID           uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserPlanType UserPlanType `gorm:"type:varchar(20);not null;default:standard;index" json:"userPlanType"`
	Name         string       `gorm:"size:100;not null" json:"name"`
	Description  string       `gorm:"type:text" json:"description"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Price            int64     `gorm:"type:bigint;not null;default:0" json:"price"`
	Slots            int       `gorm:"not null;default:0" json:"slots"`
	LastSlotsResetAt time.Time `json:"lastSlotsResetAt"`

	Subscriptions []Subscription `gorm:"foreignKey:UserPlanID" json:"subscriptions,omitempty"`
}
